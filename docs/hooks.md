# Optional Git hook composition

cidx does not install or overwrite Git hooks. If you already maintain a
post-commit hook, you may compose this local command into it:

```sh
cidx index --reason commit
```

Alternatively, use your existing `core.hooksPath` workflow and place that call
where it fits. Decide the failure behavior yourself: an indexing failure after
a successful commit cannot undo the commit, so many projects log the failure
and continue while others notify CI or a developer.

The command indexes the live worktree when the hook runs, not the committed
HEAD blobs. After a partial commit it can therefore include remaining
uncommitted and untracked non-ignored source files. Do not use it when that
live-worktree behavior conflicts with your repository policy.
