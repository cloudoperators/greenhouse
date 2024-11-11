## Greenhouse documentation

This directory contains the documentation for Greenhouse, the PlusOne operations platform.

All directories containing an `_index.md` with the following content are synchronized to the website.

```markdown
---
title: "<title>"
linkTitle: "<link>"
landingSectionIndex: <true|false>
description: >
  <Long description of the content>
---
```

You can execute the following command to serve the documentation locally:

```bash
make serve-docs
```
