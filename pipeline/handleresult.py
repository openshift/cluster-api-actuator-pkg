#!/usr/bin/env python3
import xml.dom.minidom
import argparse
import codecs
import os


class TestResult:

    def __init__(self):
        self.isJenkinsEnv = "JENKINS_AGENT_NAME" in os.environ
        print("isJenkinsEnv: {0}\\n".format(str(self.isJenkinsEnv)))

    def splitRP(self, input):
        try:
            noderoot = xml.dom.minidom.parse(input)
            origintestsuite = noderoot.getElementsByTagName("testsuite")[0]
            mods = {}

            cases = noderoot.getElementsByTagName("testcase")
            for case in cases:
                failcount = 0
                skippedcount = 0
                errorcount = 0
                if len(case.getElementsByTagName("failure")) != 0:
                    failcount = 1
                if len(case.getElementsByTagName("skipped")) != 0:
                    skippedcount = 1
                name = case.getAttribute("name")
                subteam = "Cluster_Infrastructure"
                names = []
                if "[BeforeSuite]" not in name:
                    tmpname = name.replace("'", "")
                    names.append(tmpname)
                    casedesc = {"case": case, "names": names}
                    mod = mods.get(subteam)
                    # adjust to that tests is the total case, skipped is skipped case, and failures is only failure case.
                    if mod is not None:
                        mod["cases"].append(casedesc)
                        mod["tests"] = mod["tests"] + 1
                        mod["skipped"] = mod["skipped"] + skippedcount
                        mod["failure"] = mod["failure"] + failcount
                    else:
                        mods[subteam] = {
                            "cases": [casedesc],
                            "tests": 1,
                            "skipped": skippedcount,
                            "failure": failcount,
                            "errors": errorcount,
                        }

            for k, v in mods.items():
                impl = xml.dom.minidom.getDOMImplementation()
                newdoc = impl.createDocument(None, None, None)
                testsuite = newdoc.createElement("testsuite")
                testscount = v["tests"]
                failurescount = v["failure"]
                skippedcount = v["skipped"]
                errorcount = v["errors"]
                testsuite.setAttribute(
                    "time", origintestsuite.getAttribute("time")
                )  # RP does not depend on it
                if self.isJenkinsEnv:
                    newsuitename = k
                else:
                    newsuitename = k  # it will change to other after we get testsuite name rule for sippy.
                testsuite.setAttribute("name", newsuitename)

                for case in v["cases"]:
                    testnum = 0
                    failnum = 0
                    skipnum = 0
                    result = "PASS"
                    if len(case["case"].getElementsByTagName("skipped")) != 0:
                        result = "SKIP"
                    if len(case["case"].getElementsByTagName("failure")) != 0:
                        result = "FAIL"

                    for name in case["names"]:
                        if result == "PASS":
                            testnum = testnum + 1
                        if result == "FAIL":
                            testnum = testnum + 1
                            failnum = failnum + 1
                        if result == "SKIP":
                            skipnum = skipnum + 1
                            testnum = testnum + 1
                        dupcase = case["case"].cloneNode(True)
                        dupcase.setAttribute("name", name)
                        testsuite.appendChild(dupcase)

                    if testnum > 0:
                        testnum = testnum - 1
                    if failnum > 0:
                        failnum = failnum - 1
                    if skipnum > 0:
                        skipnum = skipnum - 1
                    testscount = testscount + testnum
                    failurescount = failurescount + failnum
                    skippedcount = skippedcount + skipnum

                testsuite.setAttribute("tests", str(testscount))
                testsuite.setAttribute("failures", str(failurescount))
                testsuite.setAttribute("skipped", str(skippedcount))
                testsuite.setAttribute("errors", str(errorcount))
                newdoc.appendChild(testsuite)

                with open("import-" + k + ".xml", "wb+") as f:
                    writer = codecs.lookup("utf-8")[3](f)
                    newdoc.writexml(writer, encoding="utf-8")
                    writer.close()
        except FileNotFoundError:
            print(f"Input file {input} is not present, please check!")
        except Exception as e:
            print(f"Error occurred while parsing the file {input}: {e}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser("handleresult.py")
    parser.add_argument(
        "-a", "--action", default="get", choices={"split"}, required=True
    )
    parser.add_argument("-i", "--input", default="")
    args = parser.parse_args()

    testresult = TestResult()
    if args.action == "split":
        testresult.splitRP(args.input)
        exit(0)
