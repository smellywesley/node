# /instinct-status

Synchronize privacy-safe session metadata, then show project and global
instincts:

```powershell
scripts\learning.cmd sync
scripts\learning.cmd status
```

Treat project instincts below confidence `0.7` as suggestions. Accepted product
decisions and repository invariants take precedence over learned behavior.
