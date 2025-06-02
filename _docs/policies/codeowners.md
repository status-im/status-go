# Code Owners Policy

### Purpose

This policy outlines how code ownership is assigned and ensures that core contributors (CCs) are accountable for 
reviewing and maintaining code in the `status-go` repository.

### Code Owners File

- Code owners MUST be stored in the `.github/CODEOWNERS` file, as per GitHub's [CODEOWNERS documentation](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners).

### Assigning Code Ownership

1. Eligibility
 
- Only CCs MAY be code owners. External contributors are not eligible to be code owners.

2. Nomination
 
- For **existing** code:
  - Any CC MAY nominate themselves as a code owner for any part of the existing code.
  - Nominations SHOULD be discussed and agreed upon with other CCs to prevent and address overlaps or concerns.
- For **new** code:
  - When introducing a new module or substantial new component, CCs SHOULD assign ownership.
  - File-level ownership MAY be used where appropriate but is not required.

### Pull Request Reviews

- At least one of the code owners for the affected files MUST review a PR for it to be eligible for merging.
- If a code owner is unresponsive within a reasonable timeframe, another CC from the relevant area may approve 
the PR in their place.
