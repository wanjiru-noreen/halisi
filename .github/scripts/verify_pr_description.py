#!/usr/bin/env python3
"""
verify_pr_description.py
Validates GitHub Pull Request metadata, title conventions, description fulfillment,
and branch targeting rules for the Halisi repository.
"""

import json
import os
import re
import sys


def clean_body_text(body: str) -> str:
    """Strip HTML comments and whitespace to measure substantive user content."""
    # Remove HTML comments <!-- ... --> (non-greedy across lines)
    no_comments = re.sub(r"<!--.*?-->", "", body, flags=re.DOTALL)
    # Remove markdown headers and whitespace
    return no_comments.strip()


def validate_pull_request(event_data: dict) -> list[str]:
    errors = []
    pr = event_data.get("pull_request", {})

    title = pr.get("title", "").strip()
    body = pr.get("body") or ""
    base_ref = pr.get("base", {}).get("ref", "")
    head_ref = pr.get("head", {}).get("ref", "")
    is_draft = pr.get("draft", False)

    print("==================================================")
    print("🔍 Pull Request Compliance & Description Verification")
    print("==================================================")
    print(f"Title:    {title}")
    print(f"Base:     {base_ref}")
    print(f"Head:     {head_ref}")
    print(f"Draft:    {is_draft}")
    print("--------------------------------------------------")

    # 1. Title format validation (Conventional Commits)
    conventional_regex = (
        r"^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)"
        r"(\([a-zA-Z0-9_\-\./]+\))?!?: .+$"
    )
    if not re.match(conventional_regex, title):
        errors.append(
            "PR title does not adhere to Conventional Commits standard.\n"
            "   Format: '<type>(<scope>): <short description>'\n"
            "   Allowed types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert\n"
            "   Example: 'feat(ci): add GitHub Actions CI workflow and PR check'"
        )
    else:
        print("✅ PR title follows Conventional Commits format.")

    # 2. Branch targeting rules
    if base_ref == "main":
        if head_ref != "develop":
            errors.append(
                f"PR targets 'main' from '{head_ref}'.\n"
                "   Direct feature merges into 'main' are forbidden.\n"
                "   Only verified releases originating from 'develop' may target 'main'."
            )
        else:
            print("✅ PR targets 'main' from 'develop' (Release flow).")
    elif base_ref == "develop":
        print("✅ PR targets 'develop' (Development flow).")

    # 3. Description substance & length
    substantive_content = clean_body_text(body)
    # Remove checkboxes and template labels to ensure actual user writing
    substantive_text_only = re.sub(r"- \[[ xX]\][^\n]*", "", substantive_content)
    substantive_text_only = re.sub(r"#[^\n]*", "", substantive_text_only).strip()

    if len(substantive_text_only) < 30:
        errors.append(
            f"PR description is too brief or unmodified ({len(substantive_text_only)} non-comment characters).\n"
            "   Please provide a substantive summary of your changes explaining what was changed and why."
        )
    else:
        print("✅ PR description contains substantial explanatory content.")

    # 4. Check for incomplete placeholder 'Closes #'
    if re.search(r"Closes\s+#\s*($|\n|[\r\n]+-)", body, re.IGNORECASE):
        errors.append(
            "Unfilled issue placeholder detected: 'Closes #' without an issue number.\n"
            "   Please replace it with the actual issue number (e.g. 'Closes #12')."
        )

    # 5. Issue link check for develop PRs
    if base_ref == "develop":
        issue_link_pattern = r"(Closes|Fixes|Resolves)\s+#[0-9]+"
        if not re.search(issue_link_pattern, body, re.IGNORECASE):
            errors.append(
                "No linked issue reference found in PR description.\n"
                "   Pull requests targeting 'develop' must link a tracking issue.\n"
                "   Example: 'Closes #12' or 'Fixes #9'."
            )
        else:
            print("✅ Valid issue link reference detected.")

    # 6. Type of change / Checkbox selection
    checkbox_selected = bool(re.search(r"^\s*-\s*\[[xX]\]\s+.+", body, re.MULTILINE))
    if not checkbox_selected:
        errors.append(
            "No checklist items selected in the PR description.\n"
            "   Please select at least one item under 'Type of Change' or 'Pre-Merge Checklist' (e.g., '- [x] 🚀 New feature')."
        )
    else:
        print("✅ Checklist selections verified.")

    return errors


def write_step_summary(errors: list[str], event_data: dict) -> None:
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if not summary_path:
        return

    pr = event_data.get("pull_request", {})
    title = pr.get("title", "")
    base_ref = pr.get("base", {}).get("ref", "")

    try:
        with open(summary_path, "a", encoding="utf-8") as f:
            f.write("### 📋 PR Compliance & Description Check\n\n")
            f.write(f"- **Title**: `{title}`\n")
            f.write(f"- **Target Branch**: `{base_ref}`\n\n")

            if errors:
                f.write("#### ❌ Requirements Not Met\n\n")
                for err in errors:
                    f.write(f"- {err}\n")
                f.write("\n> 💡 Please update your PR title and description to satisfy the checklist.\n")
            else:
                f.write("#### ✅ All Compliance Checks Passed\n\n")
                f.write("PR title, description substance, linked issue, and checklists are fulfilled.\n")
    except Exception as e:
        print(f"Warning: Failed to write GITHUB_STEP_SUMMARY: {e}", file=sys.stderr)


def main() -> int:
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    if len(sys.argv) > 1:
        event_path = sys.argv[1]

    if not event_path or not os.path.exists(event_path):
        print(
            f"❌ Error: Event payload file not found: {event_path}",
            file=sys.stderr,
        )
        return 1

    with open(event_path, "r", encoding="utf-8") as f:
        event_data = json.load(f)

    errors = validate_pull_request(event_data)
    write_step_summary(errors, event_data)

    print("--------------------------------------------------")
    if errors:
        print(f"❌ PR verification FAILED with {len(errors)} error(s):")
        for idx, err in enumerate(errors, 1):
            print(f"  {idx}. {err}")
        return 1

    print("🎉 PR title and description fulfillment verification PASSED successfully!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
