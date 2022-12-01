package gatherer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CLI provides function to call the OpenShift CLI, which is using to simplify state gathering during tests.
// This wrapper was inspired by the openshift/origin version of a similar helper.
// Origin version https://github.com/openshift/origin/blob/1ec0eb3175f25b525abb39253528e230a9a85684/test/extended/util/client.go#L80
type CLI struct {
	execPath         string
	verb             string
	configPath       string
	namespace        string
	token            string
	outputBasePath   string
	subPath          string
	runtimeClient    runtimeclient.Client
	globalArgs       []string
	commandArgs      []string
	finalArgs        []string
	withoutNamespace bool
	stdin            *bytes.Buffer
	stdout           io.Writer
	stderr           io.Writer
	verbose          bool
}

// NewCLI initializes the OC CLI wrapper.
func NewCLI(namespace string, client runtimeclient.Client, outputBasePath string) (*CLI, error) {
	cli := &CLI{
		execPath:       "oc",
		namespace:      namespace,
		outputBasePath: outputBasePath,
		runtimeClient:  client,
		configPath:     kubeConfigPath(),
		verbose:        true,
	}

	return cli, nil
}

// KubeConfigPath returns the value of KUBECONFIG environment variable.
func kubeConfigPath() string {
	// can't use gomega in this method since it might be used outside of the tests.
	return os.Getenv("KUBECONFIG")
}

// setOutput allows to override the default command output.
func (oc *CLI) setOutput(out io.Writer) *CLI {
	oc.stdout = out
	return oc
}

func (oc *CLI) outputs(stdOutBuff, stdErrBuff *bytes.Buffer) (string, string, error) {
	cmd, err := oc.start(stdOutBuff, stdErrBuff)
	if err != nil {
		return "", "", err
	}

	err = cmd.Wait()

	stdOutBytes := stdOutBuff.Bytes()
	stdErrBytes := stdErrBuff.Bytes()
	stdOut := strings.TrimSpace(string(stdOutBytes))
	stdErr := strings.TrimSpace(string(stdErrBytes))

	var exitError *exec.ExitError

	switch {
	case err == nil:
		oc.stdout = bytes.NewBuffer(stdOutBytes)
		oc.stderr = bytes.NewBuffer(stdErrBytes)

		return stdOut, stdErr, nil
	case errors.As(err, &exitError):
		klog.Infof("Error running %v:\nStdOut>\n%s\nStdErr>\n%s\n", cmd, stdOut, stdErr)

		return stdOut, stdErr, err
	default:
		panic(fmt.Errorf("unable to execute %q: %w", oc.execPath, err))
	}
}

func (oc *CLI) start(stdOutBuff, stdErrBuff *bytes.Buffer) (*exec.Cmd, error) {
	oc.finalArgs = append(oc.globalArgs, oc.commandArgs...)
	if oc.verbose {
		klog.Infof("DEBUG: oc %s\n", oc.printCmd())
	}

	cmd := exec.Command(oc.execPath, oc.finalArgs...)
	cmd.Stdin = oc.stdin

	cmd.Stdout = stdOutBuff
	cmd.Stderr = stdErrBuff

	return cmd, cmd.Start()
}

func (oc *CLI) printCmd() string {
	return strings.Join(oc.finalArgs, " ")
}

// OutputToFile executes the command and store output to a file.
func (oc *CLI) OutputToFile(filename string) (string, error) {
	content, _, err := oc.Outputs()
	if err != nil {
		return "", err
	}

	path := filepath.Join(oc.outputBasePath, oc.subPath)
	filePath := filepath.Join(path, oc.Namespace()+"-"+filename)

	if len(content) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", fmt.Errorf("err create output directory %s: %w", path, err)
	}

	return path, os.WriteFile(filePath, []byte(content), 0600)
}

// Output executes the command and returns stdout/stderr combined into one string.
func (oc *CLI) Output() (string, error) {
	var buff bytes.Buffer
	_, _, err := oc.outputs(&buff, &buff)

	return strings.TrimSpace(buff.String()), err
}

// Outputs executes the command and returns the stdout/stderr output as separate strings.
func (oc *CLI) Outputs() (string, string, error) {
	var stdOutBuff, stdErrBuff bytes.Buffer

	return oc.outputs(&stdOutBuff, &stdErrBuff)
}

// Namespace returns the name of the namespace used in the current test case.
// If the namespace is not set, an empty string is returned.
func (oc *CLI) Namespace() string {
	return oc.namespace
}

// WithNamespace sets a new namespace.
func (oc CLI) WithNamespace(ns string) *CLI {
	oc.namespace = ns
	return &oc
}

// WithoutNamespace instructs the command should be invoked without adding --namespace parameter.
func (oc CLI) WithoutNamespace() *CLI {
	oc.withoutNamespace = true
	return &oc
}

// WithSubPath sets a sub path for files with CLI output.
func (oc CLI) WithSubPath(subPath string) *CLI {
	oc.subPath = subPath
	return &oc
}

// WithExec overrides 'oc' executable path.
func (oc CLI) WithExec(execPath string) *CLI {
	oc.execPath = execPath
	return &oc
}

// Run executes given OpenShift CLI command verb (iow. "oc <verb>").
// This function also override the default 'stdout' to redirect all output
// to a buffer and prepare the global flags such as namespace and config path.
func (oc *CLI) Run(commands ...string) *CLI {
	in, out, errout := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	nc := &CLI{
		execPath:       oc.execPath,
		verb:           commands[0],
		configPath:     oc.configPath,
		outputBasePath: oc.outputBasePath,
		subPath:        oc.subPath,
		namespace:      oc.Namespace(),
		globalArgs:     commands,
	}

	if len(oc.configPath) > 0 {
		nc.globalArgs = append([]string{fmt.Sprintf("--kubeconfig=%s", oc.configPath)}, nc.globalArgs...)
	}

	if len(oc.configPath) == 0 && len(oc.token) > 0 {
		nc.globalArgs = append([]string{fmt.Sprintf("--token=%s", oc.token)}, nc.globalArgs...)
	}

	if !oc.withoutNamespace {
		nc.globalArgs = append([]string{fmt.Sprintf("--namespace=%s", oc.Namespace())}, nc.globalArgs...)
	}

	nc.stdin, nc.stdout, nc.stderr = in, out, errout

	return nc.setOutput(oc.stdout)
}

// Args sets the additional arguments for the OpenShift CLI command.
func (oc *CLI) Args(args ...string) *CLI {
	oc.commandArgs = args
	return oc
}

// DumpPodLogs will dump any pod logs within namespace.
func (oc *CLI) DumpPodLogs(pods *corev1.PodList, extraLogArgs ...string) {
	for _, pod := range pods.Items {
		dumpContainer := func(container *corev1.Container) {
			logArgs := []string{"pod/" + pod.Name, "-c", container.Name, "-n", pod.Namespace}
			logArgs = append(logArgs, extraLogArgs...)
			logFilePath, err := oc.Run("logs").WithoutNamespace().Args(logArgs...).OutputToFile(pod.Name + "_" + container.Name)

			if err == nil {
				if logFilePath == "" {
					klog.Infof("No logs found for pod %s/%s, skipping", pod.Name, container.Name)
				}
			} else {
				klog.Errorf("Error retrieving logs for pod %q/%q: %v\n\n", pod.Name, container.Name, err)
			}
		}

		for _, c := range pod.Spec.InitContainers {
			dumpContainer(&c)
		}

		for _, c := range pod.Spec.Containers {
			dumpContainer(&c)
		}
	}
}

// DumpPodLogsSinceTime will dump any pod logs within namespace from the time provided.
func (oc *CLI) DumpPodLogsSinceTime(ctx context.Context, sinceTime time.Time) {
	pods := &corev1.PodList{}

	if err := oc.runtimeClient.List(ctx, pods, &runtimeclient.ListOptions{Namespace: oc.Namespace()}); err != nil {
		klog.Errorf("Error listing pods: %v", err)
		return
	}

	if len(pods.Items) > 0 {
		oc.DumpPodLogs(pods, "--since-time="+sinceTime.Format(time.RFC3339))
	}
}
