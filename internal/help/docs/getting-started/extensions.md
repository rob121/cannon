# Working With Extensions

Extensions are executable processes Cannon starts with `--site` and `--socket` flags.

## Capabilities

Each extension reports handlers from `GET /capabilities`, including optional **admin** and **help** endpoints.

## Registry

Use **System → Extensions** to activate extensions, restart processes, and inspect reported capabilities.

Extension-provided help articles appear under the **Extensions** group in this Help area when the process is running and exposes a `help` capability.
