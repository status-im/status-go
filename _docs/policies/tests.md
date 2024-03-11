# `status-go` Test Policy

- [Creating Tests](#creating-tests)
- [Flaky Tests](#flaky-tests)

## Creating Tests

- All new functionality MUST be introduced with tests that:
  - Prove that the functionality performs as described
  - Can be falsified
  - Are resistant to fuzzing
- All new tests MUST BE validated via 1000 local tests. Ensuring that the test runs and passes consistently every time gives confidence that the test is not flaky.
    - add the `-count` flag to the test command eg: `-count 1000`
    - OR wrap your test in a for loop
        ```go
        func TestTheThing(t *testing.T) {
            for i := 1; i < 1000; i++ {
                fmt.Println("Test Run", i)
                // your test body goes in here
            }
        }
        ```
## Flaky Tests

- All flaky tests / failing tests must be resolved
  - Do not introduce flaky tests to the codebase
- Steps to resolving flaky test
  - Ensure that the test is definitely flaky and that it wasn’t you that introduced a bug that made the test flaky.
      - Is a new test you’ve written flaky?
        - Ok, you need to fix that before merge is acceptable.
      - Has an old test become flaky?
        - Ok, did you touch this test or the functionality it tests? If yes, you need to fix that before merge is acceptable. Sorry.
  - If an old test fails and seems flaky either locally or in CI, you must check the `status-go` GitHub repo issues for the test name(s) failing.
      - If the test appears in the list of flaky test issues
          - If the issue is open
              - Add a comment to the issue
              - Detail that you have experienced the test being flaky and in what context (local vs CI, link to the PR or branch).
          - If the issue is closed
              - Reopen the issue OR create a new issue referencing the previous issue
                - Either is fine, use your best judgement in this case.
              - Detail that you have experienced the test being flaky and in what context (local vs CI, link to the PR or branch).
              - I just added this to see if you read the details of this PR `status-go` label for use `E:Flaky Test` may want to be not an epic, but I don't know
      - If the test does not appear in the list of flaky test issues
          - create a new issue
              - The issue title should include the flaky test name
              - The issue should use the `E:Flaky Test` label, see here for all flaky tests https://github.com/status-im/status-go/labels/E%3AFlaky%20Test
          - Detail that you have experienced the test being flaky and in what context (local vs CI, link to the PR or branch).
