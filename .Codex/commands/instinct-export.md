# /instinct-export

Export only reviewable project instincts, never raw observations:

```powershell
scripts\learning.cmd sync
scripts\learning.cmd export
```

The default artifact is `outputs/agentos-instincts.md`. A relative output path
may be supplied as the second argument.
