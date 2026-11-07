# 🚀 Production Release: develop ➔ main

## 📦 Release Overview
<!-- High-level summary of the features, optimizations, and bug fixes included in this release. -->

## 🏷️ Release Metadata
* **Target Version / Tag**: `v1.x.x`
* **Target Milestone(s)**: 

## 📋 Included Issues & Pull Requests
<!-- List all issues and feature PRs being released into main. -->
- Closes #
- Closes #

## 🛡️ Pre-Release Acceptance Checklist
- [ ] All features and bugfixes in this batch have been merged and tested on `develop`.
- [ ] Automated CI pipeline (`make check`) passes with zero warnings.
- [ ] Database migrations have been tested against a staging database (both up and down).
- [ ] Environment variable changes (`.env.example`) have been verified and applied to production.
- [ ] Security vulnerability scan (`make security`) passed with zero high/critical issues.
- [ ] Release changelog and release notes have been prepared.

## ⚠️ Rollback & Contingency Plan
<!-- Outline the rollback strategy if an issue arises post-deployment. -->
1. Revert release commit on `main`.
2. Redeploy previous stable container tag.
3. Roll back database migrations using `make migrate-down` if necessary.
