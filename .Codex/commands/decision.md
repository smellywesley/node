# /decision <topic>

Create a durable architectural or product decision.

1. Gather context, constraints, considered options, and evidence.
2. Choose the next sequence number for today's records.
3. Copy `data/templates/decision.md` to
   `data/decisions/YYYY-MM-DD-NNN-<slug>.md`.
4. Record status, decision, consequences, owner, and review date.
5. Link the decision from `data/projects/current.md`.
6. Append a `decision_recorded` event to the session ledger.

Do not use a decision record for transient notes or an unmade choice.
