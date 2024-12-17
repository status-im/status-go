# Purpose

Policy Zero establishes the foundational guidelines for creating,
reviewing, and maintaining policies in the `status-go` Git repository.
This policy aims to create a collaborative, inclusive, and transparent
process for defining repository policies, specifically regarding how
developers engage with and contribute to the `status-go` repository.

# Submitting a Policy Proposal

- Any individual MAY propose a new policy.
- Policy ideas SHOULD be discussed with contributors who wish to have
a voice in how the repository operates, including Core Contributors
(CCs) and external contributors.
- All policies MUST be submitted to the `_docs/policies`
 directory as a pull request (PR) within the `status-go` repository.
- All policies MUST be in Markdown format.
- Policy file names MUST follow [File name conventions for ADRs](https://github.com/joelparkerhenderson/architecture-decision-record?tab=readme-ov-file#file-name-conventions-for-adrs), e.g. `submitting-policy.md`.

# Review and Approval Process

The core function of the review and approval process for policy
PRs is to reach consensus on any issue and to reflect the range
of perspectives among all contributors who wish to have a voice
in how the `status-go` repository operates, including both Core
Contributors (CCs) and external contributors. Policy submissions
must aim to achieve broad contributor support and give key stakeholders
a chance to gain context of the policy requirements.

- Policy PRs SHOULD be reviewed by as many contributors as possible
who wish to engage in the process.
- Any CC MAY review, approve, and/or request changes to a policy
proposal PR.
- For any policy PR to be eligible for merging, it:
  - MUST address all feedback provided during the review process.
  - MUST be approved by all team leads (of Status Desktop and Mobile).
  - MUST be approved by all members of the @status-im/status-go-guild
  GitHub team.
  - MUST receive a minimum of six approvals from CCs.

# Policy Overrides

On rare occasions, circumstances may necessitate that an established
policy is circumvented when merging a feature PR. This is considered
an **override** and MUST follow the process outlined below to ensure
transparency and collective agreement:

- Any override MUST be documented in textual form in the circumventing
feature PR and MUST include:
  - The specific policy being overridden,
  - The rationale for taking this action,
  - The potential risks and impacts of the override, and
  - Steps taken to minimise those risks.
- Before proceeding, the override MUST be approved in writing in the
circumventing feature PR by:
  - At least one team lead from the Status Desktop or Mobile teams, AND
  - At least one member of the @status-im/status-go-guild GitHub team.
- In exceptional circumstances if an override MUST be executed
immediately due to urgency, the action SHOULD be documented as soon
as possible, and retrospective approval MUST be sought and recorded
at the earliest possible opportunity.
- Policies MAY define additional rules for handling overrides, provided
these baseline requirements are also met.

# Policy Amendments and Archival

Policies can be amended or archived to ensure they remain relevant
and aligned with contributor needs.

- Amendments
  - Policies MAY be amended at any time.
  - Amendments MUST be submitted via a PR to the `status-go` repository.
- Archival
  - Policies MAY be archived if they are obsolete or replaced by
  newer policies.
  - Archival MUST be performed by submitting a PR that moves the
  policy to `_docs/policies/archived`.
- The PR MUST include a rationale for the proposed amendment or
archival in the PR description.
- The PR MUST follow the [Review and Approval Process](#review-and-approval-process).
