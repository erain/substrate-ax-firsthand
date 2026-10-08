# Publication status and article checklist

The companion repository is public at [erain/substrate-ax-firsthand](https://github.com/erain/substrate-ax-firsthand), on the `main` branch under Apache-2.0. The README and article link to that repository and its ZIP download.

The article remains a draft. No Substack or LinkedIn post has been published as part of this work.

## Editorial choices made

- Working title: **So you want to get started with Agent Substrate and AX?**
- Audience: developers comfortable with containers and basic Kubernetes.
- Entry point: Docker running and modern Go installed; ZIP download avoids requiring host Git. Setup is in the article, not a prerequisite reading detour.
- Promise: an end-to-end path, with 10–20 minutes of experiments plus measured, clearly qualified initial setup/build time. Do not promise an unmeasured cold laptop-to-cluster duration.
- Hook: ask whether every idle agent session needs a live Pod. Connect that platform-design question to Substrate's actor/capacity split and AX's declarative API, then prove full-state restoration with two counters and contrast AX's workspace/data restoration.
- No API keys, model charges, autonomous-loop claims, density benchmarks, or production-hardening claims.
- Local Linux/amd64 edition. A custom runner keeps the first task independent of Git downloads, workspace goals, and model setup.

## Before publishing on Substack

1. Verify `go run -buildvcs=false ./cmd/lab setup` from a fresh ZIP extraction on a Linux/amd64 host with only Docker and Go as developer tooling. Also run the documented wrappers, request, router, smoke test, cleanup, and retry path. The current validation includes a fresh cluster and source/Go caches, but not a completely cold Docker host. Record cold-cache limitations and setup duration in `docs/VALIDATION.md` without turning one measurement into a universal timing promise.
2. Have another developer follow the manual steps, including predictions and expected output, without help. Time only the defined hands-on portion; shorten the article if it exceeds 20 minutes.
3. Review the AX patch against the pinned upstream API. Decide whether to retain the explicit tested patch or update the pair and re-test. Do not quietly switch to latest.
4. Keep checking tracked files for tokens, kubeconfigs, personal paths, generated manifests, binary artifacts, and registry auth when updating the samples. `.cache/`, `.local/`, and `bin/` must remain untracked. No model credential belongs in this repo.
5. Confirm both source links and companion links resolve. Compare every article command with the validated README. In Substack, check code blocks, line wrapping, quotes, table rendering, and that explanatory YAML isn't presented as copy-and-run input.
6. Review claims against observed results: same Pod does not establish every actor's placement or performance; DATA is not RAM restoration; Task Running is not task success. Retain the setup-time and early-stage caveats.
7. Keep the public repository available before publishing the article; readers need the generated assets and setup scripts, not just copied excerpts.

Tag the tested companion release and consider linking that tag's setup guide from the article so future sample changes do not silently change an already-published walkthrough.

## Suggested Substack presentation

- Subtitle: “Two hands-on experiments: pause a process that remembers, then resume a task whose workspace survives.”
- Use the two prediction prompts as short pause points. They are the learning path, not incidental prose.
- Optional lead screenshot: terminal output showing `1/1 → 2/2 → 3/3` beside the unchanged worker UID. Capture a real test; redact host/user details.
- Keep deep snapshot internals, credential injection, egress policy, model billing, and the bounded agent loop for follow-up articles.
- LinkedIn adaptation is explicitly outside this preparation task.
