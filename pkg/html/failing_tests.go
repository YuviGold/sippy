package html

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
	"k8s.io/klog"
)

func summaryTopFailingTestsWithBug(topFailingTestsWithBug, prevTopFailingTestsWithBug []sippyprocessingv1.FailingTestResult, endDay int, release string) string {
	// test name | bug | pass rate | higher/lower | pass rate
	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center"><a class="text-dark" title="Most frequently failing tests with a known bug, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a bug against the test." id="TopFailingTestsWithABug" href="#TopFailingTestsWithABug">Top Failing Tests With A Bug</a></th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>File a Bug</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, endDay)

	s += topFailingTestsRows(topFailingTestsWithBug, prevTopFailingTestsWithBug, endDay, release)

	s = s + "</table>"

	return s
}

func summaryTopFailingTestsWithoutBug(topFailingTestsWithBug, prevTopFailingTestsWithBug []sippyprocessingv1.FailingTestResult, endDay int, release string) string {
	// test name | bug | pass rate | higher/lower | pass rate
	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center"><a class="text-dark" title="Most frequently failing tests without a known bug, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a bug against the test." id="TopFailingTestsWithoutABug" href="#TopFailingTestsWithoutABug">Top Failing Tests Without A Bug</a></th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>File a Bug</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, endDay)

	s += topFailingTestsRows(topFailingTestsWithBug, prevTopFailingTestsWithBug, endDay, release)

	s = s + "</table>"

	return s
}

func topFailingTestsRows(topFailingTests, prevTopFailingTests []sippyprocessingv1.FailingTestResult, endDay int, release string) string {
	// test name | bug | pass rate | higher/lower | pass rate
	s := ""

	template := `
		<tr>
			<td>%s</td><td>%s</td><td>%0.2f%% <span class="text-nowrap">(%d runs)</span></td><td>%s</td><td>%0.2f%% <span class="text-nowrap">(%d runs)</span></td>
		</tr>
	`
	naTemplate := `
		<tr>
			<td>%s</td><td>%s</td><td>%0.2f%% <span class="text-nowrap">(%d runs)</span></td><td/><td>NA</td>
		</tr>
	`

	count := 0
	for _, testResult := range topFailingTests {
		// if we only have one failure, don't show it on the glass.  Keep it in the actual data so we can choose how to handle it,
		// but don't bother creating the noise in the UI for a one-off/long tail.
		if (testResult.TestResultAcrossAllJobs.Failures + testResult.TestResultAcrossAllJobs.Flakes) == 1 {
			continue
		}
		count++
		if count > 20 {
			break
		}

		encodedTestName := url.QueryEscape(regexp.QuoteMeta(testResult.TestName))

		testLink := fmt.Sprintf("<a target=\"_blank\" href=\"https://search.ci.openshift.org/?maxAge=168h&context=1&type=bug%%2Bjunit&name=%s&maxMatches=5&maxBytes=20971520&groupBy=job&search=%s\">%s</a>", release, encodedTestName, testResult.TestName)

		var testPrev *sippyprocessingv1.FailingTestResult
		for _, prevTestResult := range prevTopFailingTests {
			if prevTestResult.TestName == testResult.TestName {
				testPrev = &prevTestResult
				break
			}
		}

		klog.V(2).Infof("processing top failing tests %s, bugs: %v", testResult.TestName, testResult.TestResultAcrossAllJobs.BugList)
		bugHTML := bugHTMLForTest(testResult.TestResultAcrossAllJobs.BugList, release, "", testResult.TestResultAcrossAllJobs.Name)
		if testPrev != nil {
			arrow := ""
			delta := 5.0
			if testResult.TestResultAcrossAllJobs.Successes+testResult.TestResultAcrossAllJobs.Failures > 80 {
				delta = 2
			}
			if testResult.TestResultAcrossAllJobs.PassPercentage > testPrev.TestResultAcrossAllJobs.PassPercentage+delta {
				arrow = up
			} else if testResult.TestResultAcrossAllJobs.PassPercentage < testPrev.TestResultAcrossAllJobs.PassPercentage-delta {
				arrow = down
			}

			if testResult.TestResultAcrossAllJobs.PassPercentage > testPrev.TestResultAcrossAllJobs.PassPercentage+delta {
				arrow = fmt.Sprintf(up, testResult.TestResultAcrossAllJobs.PassPercentage-testPrev.TestResultAcrossAllJobs.PassPercentage)
			} else if testResult.TestResultAcrossAllJobs.PassPercentage < testPrev.TestResultAcrossAllJobs.PassPercentage-delta {
				arrow = fmt.Sprintf(down, testPrev.TestResultAcrossAllJobs.PassPercentage-testResult.TestResultAcrossAllJobs.PassPercentage)
			} else if testResult.TestResultAcrossAllJobs.PassPercentage > testPrev.TestResultAcrossAllJobs.PassPercentage {
				arrow = fmt.Sprintf(flatup, testResult.TestResultAcrossAllJobs.PassPercentage-testPrev.TestResultAcrossAllJobs.PassPercentage)
			} else {
				arrow = fmt.Sprintf(flatdown, testPrev.TestResultAcrossAllJobs.PassPercentage-testResult.TestResultAcrossAllJobs.PassPercentage)
			}

			s += fmt.Sprintf(template, testLink, bugHTML, testResult.TestResultAcrossAllJobs.PassPercentage, testResult.TestResultAcrossAllJobs.Successes+testResult.TestResultAcrossAllJobs.Failures, arrow, testPrev.TestResultAcrossAllJobs.PassPercentage, testPrev.TestResultAcrossAllJobs.Successes+testPrev.TestResultAcrossAllJobs.Failures)
		} else {
			s += fmt.Sprintf(naTemplate, testLink, bugHTML, testResult.TestResultAcrossAllJobs.PassPercentage, testResult.TestResultAcrossAllJobs.Successes+testResult.TestResultAcrossAllJobs.Failures)
		}

		collapseName := "test-result---" + testResult.TestName
		collapseName = strings.ReplaceAll(strings.ReplaceAll(collapseName, ".", ""), " ", "")
		// 1 encoded job name
		// 2 indent depth
		// 3 test name
		// 4 job name regex
		// 5 encoded test name
		// 6 bug list/bug search
		// 7 pass rate
		// 8 number of runs
		const failingTestJobGroupTemplate = `
			<tr class="%s">
				<td style="padding-left:%dpx">
					<a target="_blank" href="%s">%s</a>
				</td>
				<td>
					%0.2f%%<span class="text-nowrap">(%d runs)</span>
				</td>
				<td>
					%s
				</td>
				<td>
					%0.2f%% (%0.2f%%)<span class="text-nowrap">(%d runs)</span>
				</td>
			</tr>
	`
		const failingTestJobGroupTemplateNA = `
			<tr class="%s">
				<td style="padding-left:%dpx">
					<a target="_blank" href="%s">%s</a>
				</td>
				<td>
					%0.2f%%<span class="text-nowrap">(%d runs)</span>
				</td>
				<td></td>
				<td>
					NA
				</td>
			</tr>
	`

		jobIndentDepth := 50 + 10
		count := 10
		rowCount := 0
		rows := ""
		additionalMatches := 0
		for _, failingTestJobResult := range testResult.JobResults {
			if count == 0 {
				additionalMatches++
				continue
			}
			count--

			var prevTestJobResult *sippyprocessingv1.FailingTestJobResult
			if testPrev != nil {
				for _, prevJobInstance := range testPrev.JobResults {
					if prevJobInstance.Name == failingTestJobResult.Name {
						prevTestJobResult = &prevJobInstance
						break
					}
				}
			}

			if prevTestJobResult != nil {
				rows = rows + fmt.Sprintf(failingTestJobGroupTemplate,
					collapseName,
					jobIndentDepth,
					failingTestJobResult.TestGridUrl,
					failingTestJobResult.Name,
					failingTestJobResult.PassPercentage,
					failingTestJobResult.TestSuccesses+failingTestJobResult.TestFailures,
					"arrow",
					prevTestJobResult.PassPercentage,
					prevTestJobResult.TestSuccesses+prevTestJobResult.TestFailures,
				)
			} else {
				rows = rows + fmt.Sprintf(failingTestJobGroupTemplateNA,
					collapseName,
					jobIndentDepth,
					failingTestJobResult.TestGridUrl,
					failingTestJobResult.Name,
					failingTestJobResult.PassPercentage,
					failingTestJobResult.TestSuccesses+failingTestJobResult.TestFailures,
				)
			}
			rowCount++
		}

		if additionalMatches > 0 {
			rows += fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:%dpx">Plus %d more jobs</td></tr>`, collapseName, jobIndentDepth, additionalMatches)
		}
		if rowCount > 0 {
			s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:%dpx" class="font-weight-bold">Job Name</td><td class="font-weight-bold">Job Pass Rate</td></tr>`, collapseName, jobIndentDepth)
			s = s + rows
			s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px" class="font-weight-bold"></td><td class="font-weight-bold"></td></tr>`, collapseName)
		} else {
			s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=3 style="padding-left:%dpx" class="font-weight-bold">No Jobs Matched Filters</td></tr>`, collapseName, jobIndentDepth)
		}
	}

	s = s + "</table>"
	return s
}
