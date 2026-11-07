#!/usr/bin/env python3
"""
manage_issue_dependencies.py
Automates issue dependency tracking, parent-child relationships, and PR blocking
for the Halisi repository.

Features:
1. Detects dependency declarations in issue/PR descriptions:
   - "Blocked by #X, #Y" / "blocked-by: #X"
   - "Depends on #X" / "depends-on: #X"
   - "Parent: #X" / "Parent issue: #X" / "Sub-issue of #X"
   - Markdown sections: "### Dependencies" or "### Blocked By"
   - GitHub native issue dependencies (blocked_by & parent APIs)
2. Applies or removes the 'blocked' label automatically.
3. Automatically unblocks downstream issues when a parent/blocking issue is closed.
4. Re-blocks downstream issues if a blocking issue is reopened.
5. Blocks Pull Requests that attempt to close a blocked issue or are blocked by an open issue.
"""

import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, List, Optional, Set, Tuple

BLOCKED_LABEL = "blocked"
BLOCKED_LABEL_COLOR = "b60205"  # Deep red
BLOCKED_LABEL_DESC = "Blocked by unresolved parent or dependency issues"


def clean_body(text: Optional[str]) -> str:
    """Strip HTML comments and carriage returns."""
    if not text:
        return ""
    no_comments = re.sub(r"<!--.*?-->", "", text, flags=re.DOTALL)
    return no_comments.replace("\r\n", "\n")


def parse_dependencies_from_text(text: Optional[str], self_number: Optional[int] = None) -> List[int]:
    """
    Extracts blocking/parent issue numbers from body text.
    Supports formats:
      - Blocked by #12, #14
      - blocked-by: #12
      - Depends on #15
      - depends-on: #15
      - Parent: #10 / Sub-issue of #10
      - Full URLs: https://github.com/owner/repo/issues/12
      - Markdown lists under '### Dependencies' or '### Blocked By'
    """
    body = clean_body(text)
    if not body:
        return []

    found: Set[int] = set()

    # 1. Keyword lines (blocked by, depends on, parent, sub-issue of)
    keyword_patterns = [
        r"(?:blocked\s*by|blocked-by|depends\s*on|depends-on|parent(?:\s*issue)?|sub-issue\s*of)\s*[:\-]?\s*([^\n\r]+)",
    ]

    for pattern in keyword_patterns:
        for match in re.finditer(pattern, body, re.IGNORECASE):
            line_val = match.group(1)
            # Find all numbers like #123, 123, or issue URLs
            url_matches = re.findall(r"/issues/([0-9]+)", line_val)
            for m in url_matches:
                found.add(int(m))

            issue_matches = re.findall(r"#([0-9]+)", line_val)
            for m in issue_matches:
                found.add(int(m))

    # 2. Section blocks (### Dependencies, ### Blocked By)
    section_pattern = r"###+\s*(?:Dependencies|Blocked\s*By)[^\n]*\n([\s\S]*?)(?=\n###|\Z)"
    for match in re.finditer(section_pattern, body, re.IGNORECASE):
        section_content = match.group(1)
        for line in section_content.splitlines():
            line = line.strip()
            if not line or not line.startswith(("-", "*")):
                continue
            for num in re.findall(r"#([0-9]+)", line):
                found.add(int(num))
            for num in re.findall(r"/issues/([0-9]+)", line):
                found.add(int(num))

    # Never allow an issue to be blocked by itself
    if self_number is not None and self_number in found:
        found.remove(self_number)

    return sorted(list(found))


def parse_linked_issues(body: Optional[str]) -> List[int]:
    """Find issues linked to be closed by a PR (e.g. 'Closes #14')."""
    text = clean_body(body)
    if not text:
        return []

    pattern = r"(?:close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)\s+#([0-9]+)"
    matches = re.findall(pattern, text, re.IGNORECASE)
    return sorted(list({int(m) for m in matches}))


class GitHubClient:
    def __init__(self, token: Optional[str] = None, repo: Optional[str] = None):
        self.token = token or os.environ.get("GITHUB_TOKEN") or ""
        self.repo = repo or os.environ.get("GITHUB_REPOSITORY") or ""
        self.base_url = "https://api.github.com"

    def request(
        self,
        endpoint: str,
        method: str = "GET",
        data: Optional[Dict[str, Any]] = None,
    ) -> Any:
        url = endpoint if endpoint.startswith("http") else f"{self.base_url}/{endpoint.lstrip('/')}"
        headers = {
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "halisi-dependency-bot",
        }
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"

        encoded_data = None
        if data is not None:
            encoded_data = json.dumps(data).encode("utf-8")
            headers["Content-Type"] = "application/json"

        req = urllib.request.Request(url, data=encoded_data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req) as resp:
                status = resp.status
                content = resp.read().decode("utf-8")
                if not content:
                    return None
                return json.loads(content)
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return None
            body = e.read().decode("utf-8", errors="ignore")
            raise RuntimeError(f"GitHub API HTTP {e.code} error on {method} {url}: {body}") from e
        except Exception as e:
            raise RuntimeError(f"GitHub API request failed: {e}") from e

    def ensure_label(self, name: str, color: str, description: str) -> None:
        if not self.repo or not self.token:
            return
        endpoint = f"repos/{self.repo}/labels/{urllib.parse.quote(name)}"
        existing = self.request(endpoint, "GET")
        if existing is None:
            # Create label
            create_endpoint = f"repos/{self.repo}/labels"
            self.request(
                create_endpoint,
                "POST",
                {"name": name, "color": color, "description": description},
            )
            print(f"🏷️ Created repository label: '{name}'")

    def get_issue(self, issue_number: int) -> Optional[Dict[str, Any]]:
        endpoint = f"repos/{self.repo}/issues/{issue_number}"
        return self.request(endpoint, "GET")

    def get_native_dependencies(self, issue_number: int) -> List[int]:
        """Queries GitHub's native issue dependencies API (blocked_by and parent)."""
        deps: Set[int] = set()
        # 1. blocked_by dependencies
        endpoint = f"repos/{self.repo}/issues/{issue_number}/dependencies/blocked_by"
        data = self.request(endpoint, "GET")
        if isinstance(data, list):
            for item in data:
                num = item.get("number")
                if num:
                    deps.add(int(num))

        # 2. parent issue
        parent_endpoint = f"repos/{self.repo}/issues/{issue_number}/parent"
        parent_data = self.request(parent_endpoint, "GET")
        if isinstance(parent_data, dict) and parent_data.get("number"):
            deps.add(int(parent_data["number"]))

        return sorted(list(deps))

    def add_label(self, issue_number: int, label: str) -> None:
        self.ensure_label(BLOCKED_LABEL, BLOCKED_LABEL_COLOR, BLOCKED_LABEL_DESC)
        endpoint = f"repos/{self.repo}/issues/{issue_number}/labels"
        self.request(endpoint, "POST", {"labels": [label]})
        print(f"➕ Added label '{label}' to #{issue_number}")

    def remove_label(self, issue_number: int, label: str) -> None:
        endpoint = f"repos/{self.repo}/issues/{issue_number}/labels/{urllib.parse.quote(label)}"
        self.request(endpoint, "DELETE")
        print(f"➖ Removed label '{label}' from #{issue_number}")

    def add_comment(self, issue_number: int, comment: str) -> None:
        endpoint = f"repos/{self.repo}/issues/{issue_number}/comments"
        self.request(endpoint, "POST", {"body": comment})
        print(f"💬 Added comment to #{issue_number}")

    def search_issues_referencing(self, target_number: int) -> List[Dict[str, Any]]:
        """Finds open issues mentioning the target issue."""
        query = f"repo:{self.repo} is:issue is:open #{target_number}"
        endpoint = f"search/issues?q={urllib.parse.quote(query)}"
        data = self.request(endpoint, "GET")
        if isinstance(data, dict):
            return data.get("items", [])
        return []

    def get_all_open_issues(self) -> List[Dict[str, Any]]:
        """Fetch open issues for repository sweep."""
        issues = []
        page = 1
        while True:
            endpoint = f"repos/{self.repo}/issues?state=open&per_page=100&page={page}"
            batch = self.request(endpoint, "GET")
            if not batch or not isinstance(batch, list):
                break
            # Filter out pull requests
            issues.extend([i for i in batch if "pull_request" not in i])
            if len(batch) < 100:
                break
            page += 1
        return issues


def evaluate_dependencies(
    client: GitHubClient,
    issue_number: int,
    body: Optional[str],
) -> Tuple[bool, List[int], List[int]]:
    """
    Evaluates dependencies for an issue.
    Returns: (is_blocked, open_blockers, closed_blockers)
    """
    text_deps = parse_dependencies_from_text(body, self_number=issue_number)
    native_deps = client.get_native_dependencies(issue_number)
    all_dep_numbers = sorted(list(set(text_deps + native_deps)))

    open_blockers: List[int] = []
    closed_blockers: List[int] = []

    for dep_num in all_dep_numbers:
        dep_issue = client.get_issue(dep_num)
        if not dep_issue:
            continue
        state = dep_issue.get("state", "open")
        if state == "open":
            open_blockers.append(dep_num)
        else:
            closed_blockers.append(dep_num)

    is_blocked = len(open_blockers) > 0
    return is_blocked, open_blockers, closed_blockers


def update_issue_blocked_status(
    client: GitHubClient,
    issue_data: Dict[str, Any],
    notify: bool = True,
) -> bool:
    """
    Calculates blocker state for an issue and synchronizes the 'blocked' label.
    Returns True if currently blocked, False otherwise.
    """
    issue_number = issue_data["number"]
    body = issue_data.get("body", "")
    current_labels = [l.get("name") for l in issue_data.get("labels", []) if isinstance(l, dict)]
    is_currently_labeled = BLOCKED_LABEL in current_labels

    is_blocked, open_blockers, closed_blockers = evaluate_dependencies(
        client, issue_number, body
    )

    if is_blocked:
        if not is_currently_labeled:
            client.add_label(issue_number, BLOCKED_LABEL)
            if notify:
                blocker_lines = "\n".join([f"- #{num}" for num in open_blockers])
                comment = (
                    "### 🛑 Issue Blocked\n\n"
                    "This issue is marked as **blocked** by the following unresolved dependency issue(s):\n"
                    f"{blocker_lines}\n\n"
                    "Please complete and close the blocking issue(s) before proceeding with this task.\n"
                    "_This label will be removed automatically once all dependencies are closed._"
                )
                client.add_comment(issue_number, comment)
        print(f"🔒 Issue #{issue_number} is BLOCKED by {open_blockers}")
        return True
    else:
        if is_currently_labeled:
            client.remove_label(issue_number, BLOCKED_LABEL)
            if notify and (open_blockers or closed_blockers):
                comment = (
                    "### ✅ Issue Unblocked\n\n"
                    "All blocking parent/dependency issues have been resolved and closed!\n"
                    "This issue is now **unblocked** and ready for development. 🚀"
                )
                client.add_comment(issue_number, comment)
        print(f"🔓 Issue #{issue_number} is UNBLOCKED")
        return False


def handle_issue_event(client: GitHubClient, event_data: Dict[str, Any]) -> int:
    action = event_data.get("action", "")
    issue = event_data.get("issue", {})
    issue_number = issue.get("number")
    if not issue_number:
        print("No issue number in event payload.")
        return 0

    print(f"📋 Processing Issue #{issue_number} (Action: {action})")

    if action in ("opened", "edited", "reopened"):
        update_issue_blocked_status(client, issue, notify=True)

    if action in ("closed", "reopened"):
        # Find other open issues that depend on this issue
        print(f"🔄 Re-evaluating dependent issues linked to #{issue_number}...")
        candidate_issues = client.search_issues_referencing(issue_number)
        for cand in candidate_issues:
            cand_num = cand.get("number")
            if not cand_num or cand_num == issue_number:
                continue
            cand_full = client.get_issue(cand_num)
            if cand_full and cand_full.get("state") == "open":
                print(f"   -> Re-evaluating dependent Issue #{cand_num}...")
                update_issue_blocked_status(client, cand_full, notify=True)

    return 0


def handle_pull_request_event(client: GitHubClient, event_data: Dict[str, Any]) -> int:
    pr = event_data.get("pull_request", {})
    pr_number = pr.get("number")
    title = pr.get("title", "")
    body = pr.get("body", "")
    current_labels = [l.get("name") for l in pr.get("labels", []) if isinstance(l, dict)]
    is_currently_labeled = BLOCKED_LABEL in current_labels

    print(f"🔍 Checking Pull Request #{pr_number}: '{title}'")

    # 1. Direct dependencies declared on PR itself
    pr_is_blocked, pr_open_blockers, _ = evaluate_dependencies(client, pr_number, body)

    # 2. Linked issues that this PR closes (e.g. Closes #14)
    linked_issues = parse_linked_issues(body)
    blocked_linked_issues: Dict[int, List[int]] = {}

    for linked_num in linked_issues:
        linked_data = client.get_issue(linked_num)
        if not linked_data:
            continue
        is_linked_blocked, open_blockers, _ = evaluate_dependencies(
            client, linked_num, linked_data.get("body", "")
        )
        if is_linked_blocked:
            blocked_linked_issues[linked_num] = open_blockers

    all_blocking_reasons: List[str] = []

    if pr_is_blocked:
        all_blocking_reasons.append(
            f"PR #{pr_number} declares open blocking dependencies: "
            + ", ".join([f"#{n}" for n in pr_open_blockers])
        )

    for linked_num, blockers in blocked_linked_issues.items():
        all_blocking_reasons.append(
            f"PR targets closing Issue #{linked_num}, but #{linked_num} is blocked by open issue(s): "
            + ", ".join([f"#{n}" for n in blockers])
        )

    summary_file = os.environ.get("GITHUB_STEP_SUMMARY")

    if all_blocking_reasons:
        if not is_currently_labeled:
            client.add_label(pr_number, BLOCKED_LABEL)
        print("❌ PULL REQUEST IS BLOCKED:")
        for r in all_blocking_reasons:
            print(f"   - {r}")

        if summary_file:
            with open(summary_file, "a", encoding="utf-8") as f:
                f.write("### 🛑 Pull Request Blocked by Dependencies\n\n")
                f.write("This pull request cannot be merged because unresolved parent/dependency issues remain open:\n\n")
                for r in all_blocking_reasons:
                    f.write(f"- ⚠️ {r}\n")
                f.write("\n> 💡 Please ensure parent issues are fully completed and closed before merging dependent PRs.\n")

        return 1
    else:
        if is_currently_labeled:
            client.remove_label(pr_number, BLOCKED_LABEL)
        print("✅ No unresolved issue dependencies found. Pull request is unblocked.")

        if summary_file:
            with open(summary_file, "a", encoding="utf-8") as f:
                f.write("### ✅ Issue Dependencies Check\n\n")
                f.write("No blocking parent or dependency issues detected. PR is unblocked and clear for review/merge.\n")

        return 0


def handle_sweep(client: GitHubClient) -> int:
    print("🧹 Running full repository sweep on all open issues...")
    issues = client.get_all_open_issues()
    print(f"Found {len(issues)} open issue(s). Synchronizing blocked labels...")
    for issue in issues:
        update_issue_blocked_status(client, issue, notify=False)
    print("✅ Sweep complete.")
    return 0


def main() -> int:
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    if len(sys.argv) > 1 and not sys.argv[1].startswith("--"):
        event_path = sys.argv[1]

    client = GitHubClient()

    if "--sweep" in sys.argv:
        return handle_sweep(client)

    if not event_path or not os.path.exists(event_path):
        print("No event path provided or file does not exist. Running sweep mode.")
        return handle_sweep(client)

    with open(event_path, "r", encoding="utf-8") as f:
        event_data = json.load(f)

    event_name = os.environ.get("GITHUB_EVENT_NAME") or ""
    if not event_name:
        if "issue" in event_data:
            event_name = "issues"
        elif "pull_request" in event_data:
            event_name = "pull_request"

    if event_name == "issues":
        return handle_issue_event(client, event_data)
    elif event_name in ("pull_request", "pull_request_target"):
        return handle_pull_request_event(client, event_data)
    else:
        return handle_sweep(client)


if __name__ == "__main__":
    sys.exit(main())
