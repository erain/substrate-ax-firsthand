# Publication status and article checklist

The companion repository is public at [erain/substrate-ax-firsthand](https://github.com/erain/substrate-ax-firsthand), on the `main` branch under Apache-2.0.

The article is published on [Substack](https://yiyu9.substack.com/p/so-you-want-to-get-started-with-agent).

`article/substack.md` is the single source for the repository's tutorial. The root `README.md` is a relative symlink to it, so edit the article instead of maintaining two walkthroughs. Test commands and setup details live in `docs/SETUP.md`. Repository edits do not update the published Substack post automatically.

Keep the counter section's `1-try-the-counter` custom anchor: the published article links to that section of the old README.

## Editorial choices made

- Working title: **So you want to get started with Agent Substrate and AX?**
- Audience: developers comfortable with containers and basic Kubernetes.
- Entry point: Docker running and modern Go installed; ZIP download avoids requiring host Git. Setup is in the article, not a prerequisite reading detour.
- Commands: enter the prepared toolbox once, then show real kubectl, kubectl ate, AX, and curl commands. Readers create the counter's WorkerPool and ActorTemplate; setup only installs the platform and tutorial namespaces. Keep context, namespace, and atespace visible. Cleanup remains guarded.
- Promise: an end-to-end path, with 10–20 minutes of experiments plus measured, clearly qualified initial setup/build time. Do not promise an unmeasured cold laptop-to-cluster duration.
- Opening: start with the choice of keeping an idle session's Pod running or rebuilding its environment later. Explain how Substrate and AX approach that problem, then get to the lab.
- Voice: write as someone explaining the commands to another developer. Keep instructions and expected output close together. Avoid slogans, repeated summaries, and announcing "aha moments." Any first-person experience should come from the actual lab work, not an invented anecdote.
- No API keys, model charges, autonomous-loop claims, density benchmarks, or production-hardening claims.
- Local Linux/amd64 edition. A custom runner keeps the first task independent of Git downloads, workspace goals, and model setup.

## Before publishing on Substack

1. Verify `go run -buildvcs=false ./cmd/lab setup` from a fresh ZIP extraction on a Linux/amd64 host with only Docker and Go as developer tooling. Enter `./bin/lab shell` in both terminals and run the full commands, including worker/template creation and golden readiness. Also check the smoke test, cleanup, and retry path. The current validation includes a fresh cluster and source/Go caches, but not a completely cold Docker host. Record cold-cache limitations and setup duration in `docs/VALIDATION.md` without turning one measurement into a universal timing promise.
2. Have another developer follow the manual steps, including predictions and expected output, without help. Time only the defined hands-on portion; shorten the article if it exceeds 20 minutes.
3. Review the AX patch against the pinned upstream API. Decide whether to retain the explicit tested patch or update the pair and re-test. Do not quietly switch to latest.
4. Keep checking tracked files for tokens, kubeconfigs, personal paths, generated manifests, binary artifacts, and registry auth when updating the samples. `.cache/`, `.local/`, and `bin/` must remain untracked. No model credential belongs in this repo.
5. Confirm both source links and companion links resolve. Check the article's commands against the validated lab flow and confirm GitHub renders the symlinked README. In Substack, check code blocks, line wrapping, quotes, and lists. Make sure explanatory YAML isn't presented as copy-and-run input. Use short lists for comparisons; tables don't copy cleanly.
6. Review claims against observed results: same Pod does not establish every actor's placement or performance; DATA is not RAM restoration; Task Running is not task success. Retain the setup-time and early-stage caveats.
7. Keep the public repository available before publishing the article; readers need the generated assets and setup scripts, not just copied excerpts.

Tag the tested companion release and consider linking that tag's setup guide from the article so future sample changes do not silently change an already-published walkthrough.

## Suggested Substack presentation

- Subtitle: “A local lab for Kubernetes users, with Docker and Go.”
- Keep the questions before the two resume checks. Give readers a moment to consider the result before showing it.
- Optional lead screenshot: terminal output showing `1/1 → 2/2 → 3/3` beside the unchanged worker UID. Capture a real test; redact host/user details.
- Keep deep snapshot internals, credential injection, egress policy, model billing, and the bounded agent loop for follow-up articles.
- LinkedIn copy can link to the published article; posting it is a separate action.
