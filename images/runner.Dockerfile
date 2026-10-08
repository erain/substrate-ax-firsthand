# Deliberately no model SDK, Git clone, or autonomous bootstrap dependency.
ARG BASE_SHELL=alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
FROM ${BASE_SHELL}
COPY bin/ax-task-runner /usr/local/bin/ax-task-runner
ENTRYPOINT ["/usr/local/bin/ax-task-runner"]
