#!/bin/bash

check_for_breaking_change() {
    commit_message=$1

    feat_part=$(echo "$commit_message" | grep -oP '^\w+\(.*\)')

    if [[ "$commit_message" =~ "${feat_part}!:" ]]; then
        return 0
    elif [[ "$commit_message" =~ "${feat_part}_:" ]]; then
      return 1
    else
        echo "Error: Neither underscore nor exclamation mark found."
        exit 1
    fi
}

# Sample git commit message
sample_commit_message="feat(abc)!: some text"

# Calling the function with the sample commit message
check_for_breaking_change "$sample_commit_message"

is_breaking_change=$?

# Output the result
echo "$is_breaking_change"

# Function to increment the semantic version
increment_semantic_version() {
    version=$1
    increment=$2

    # Splitting the version into its components
    IFS='.' read -ra version_parts <<< "$version"

    major=${version_parts[0]}
    minor=${version_parts[1]}
    patch=${version_parts[2]}

    if [ "$increment" = "minor" ]; then
        minor=$((minor + 1))
        patch=0
    elif [ "$increment" = "patch" ]; then
        patch=$((patch + 1))
    else
        echo "Error: Invalid increment type. Please choose 'minor' or 'patch'."
        exit 1
    fi

    # Constructing the new version
    new_version="$major.$minor.$patch"
    echo "$new_version"
}

extract_semantic_version() {
    version_string=$1

    # Removing the leading 'v' and any trailing information after the version number
   version=$(echo "$version_string" | grep -oP 'v(\d+\.\d+\.\d+)')

    echo "${version#v}" # Removing the leading 'v'
}

# Sample version string
git_describe_output=$(git describe --tags --abbrev=0)

# Calling the function with the sample version string
extracted_version=$(extract_semantic_version "$git_describe_output")

#sample_version="1.2.3"
increment_type="patch"

## Calling the function with the sample version and increment type
new_version=$(increment_semantic_version "$extracted_version" "$increment_type")

# Output the result
echo "New version: $new_version"

oldrev="$1"
newrev="$2"
refname="$3"

# Use git log to extract the commit logs that are going to be merged
commit_logs=$(git log --pretty=format:"%h %s" $oldrev..$newrev)

# Output the commit logs
echo "Commits being merged:"
echo "$commit_logs"
