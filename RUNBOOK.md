# Runbook: Expense Approval demo

This runbook is for presenting the demo. The [README](README.md) explains what it is and how it's built.

## What it shows

An expense approval waits for a manager, escalates when they're slow, and pays out through a flaky payment provider. Each expense is one `ExpenseApproval` Workflow. You stop the worker in the middle, approve anyway, and nothing is lost.

The demo makes three problems visible:

- **A human decision can take minutes or days, and the process waiting for it can't hold it in memory.** The Workflow waits on a durable timer, and the approver's decision arrives as a Signal that Temporal records even while no worker is running. Beats 2 to 5.
- **Slow approvers need a deadline.** If nobody decides in time, the timer fires and the request escalates to a second approver, then expires. Beat 6.
- **A payment provider that fails shouldn't lose the reimbursement or pay it twice.** A Retry Policy retries the payment with backoff, and the Workflow ID is the idempotency key. Beat 7.

## Before you start

**On tmprl-demo.cloud,** open **Expense Approval** from the catalog. The app sits behind Temporal sign-in. To stop workers during the talk, use a crashable environment: a private, time-limited copy with its own worker and Task Queue, so your outage doesn't affect anyone else.

1. On the demo's catalog page, in the **Crashable environment** panel, click **Create crashable environment**. Wait for it to show **Running**.
2. Click **Open environment**. That opens your copy of the app.

The environment deletes itself after 1 hour. **Extend +1h** adds an hour.

**To run it locally,** install Go 1.26 or later and the [Temporal CLI](https://docs.temporal.io/cli). You need no account and no API key.

## Start it

### On tmprl-demo.cloud

Open the crashable environment. A healthy start shows the **New expense** form with four example chips, an empty **My requests** list ("Nothing submitted in this session yet."), an empty **Approver inbox**, and the **Workflow code** panel showing `expense/workflow.go`. The top bar shows `task queue · expense-approval-…` with your workspace's suffix. **Temporal UI ↗** opens the Namespace in Temporal Cloud.

### Locally

In three terminals, from the repository root:

1. `make temporal`: the Temporal dev server, UI at http://localhost:8233.
2. `make worker`: the worker. This is the one you stop and start during the demo.
3. `make web`: the app at http://localhost:8080.

The top bar shows `task queue · expense-approval`. **Temporal UI ↗** opens http://localhost:8233.

The app reads Temporal settings only from environment variables, never from your CLI config file, so a default profile can't point it at a real Namespace.

Escalation happens 45 seconds after the approver is notified, and an unanswered request expires after 10 minutes. To change that, set `ESCALATE_AFTER` and `EXPIRE_AFTER` in the web app's terminal before `make web`.

## The walkthrough

Every step in the **Request journey** panel comes from the Workflow's Event History, and the **Workflow code** panel highlights the line that's running. Click a request in **My requests** to show its journey.

Thirty-second version: "This is an expense approval. It waits for a manager, escalates if they're slow, and pays out through a flaky payment provider. I'm going to stop the worker in the middle, and nothing will be lost."

### 1. A small expense pays itself

1. Click the **Team lunch · $38.50** chip, then **Submit expense**.
2. Within a few seconds the journey shows Check expense policy ("Within the $50.00 auto-approval limit"), Pay reimbursement, Notify employee, and **Auto-approved and paid**. The status reads **Paid**.
3. Say: "Each row here is an event Temporal recorded. The app doesn't keep its own state about this expense."

### 2. A bigger expense waits for a person

1. Click **Conference · $649**, then **Submit expense**.
2. The journey shows **Notify Sam Chen**, then **Waiting for Sam Chen** with an "Escalates in 0:45" countdown. The status reads **Awaiting Sam Chen**, and the request appears in the **Approver inbox**.
3. The code panel highlights the `awaitDecision(ctx, req.EscalateAfter, decided)` line.
4. Say: "The Workflow is now waiting on a durable timer. Nothing is sitting in a worker's memory; Temporal holds the timer."

Do beats 3 to 5 before the countdown reaches 0, or the request escalates first. If it does, that's beat 6; carry on from there.

### 3. Stop the worker

1. On tmprl-demo.cloud, click **Scale down workers** in the catalog's **Crashable environment** panel and wait for **Workers scaled to zero**. Locally, press `Ctrl-C` in the `make worker` terminal.
2. The app keeps working. The countdown keeps running.
3. Say: "There's no worker running this Workflow now. Watch what still works."

**Crash workers** kills the worker pods, and Kubernetes replaces them within seconds. Use **Scale down workers** for this beat so the outage lasts as long as you need.

### 4. Approve with no worker running

1. In the **Approver inbox**, type a comment if you like, and click **Approve**. A toast says "Approval recorded by Temporal".
2. The journey adds **Approved by Sam Chen**, and the wait step says "Decision received; the Workflow acts on it next". A few seconds later, the status changes to **Waiting for a worker** and a banner says **No worker is running this step.** with how long the next step has been ready.
3. Say: "The decision is a Signal. Temporal wrote it into the Workflow's history even though no worker is running. It's safe."

### 5. Bring the worker back

1. On tmprl-demo.cloud, click **Scale up workers**. Locally, run `make worker` again.
2. When the worker polls, the Workflow picks up the decision, pays, and notifies the employee. The status reads **Paid**.
3. The journey keeps a **No worker for m:ss** step: "Temporal held the request, its timers, and the decision until a worker returned. Nothing was lost." The wait step reads "Decision arrived; timer canceled".
4. Say: "The worker came back and carried on from where the Workflow was. The decision was waiting for it."
5. For proof, click **Open execution ↗**. The Event History shows `WorkflowExecutionSignaled` at the time you approved, with no worker running, then `TimerCanceled` and the payment once the worker returned. In a local run the history also shows a `WorkflowTaskTimedOut` between them: the Task first went to the stopped worker and was rescheduled.

### 6. Escalation

1. Submit another expense over $50 and under $5,000 (for example **Conference · $649** again), and don't decide.
2. When the countdown runs out, the wait step reads "No decision in time", the journey adds **Escalate to Dana Ortiz**, and a new wait starts: **Waiting for Sam Chen or Dana Ortiz**, with an "Expires in" countdown. The status reads **Escalated to Dana Ortiz**.
3. The inbox item now says "deciding as Dana Ortiz" and "Escalated · expires in …". Click **Approve** to approve it as Dana, or leave it to expire.
4. Say: "The timer fired, so the Workflow escalated. Sam can still answer too. The first valid decision wins."

### 7. A payment outage

1. Click **Client dinner · payment outage**. The chip sets **Demo · payment provider fails the first** to 3 attempts. Click **Submit expense**, then **Approve** it in the inbox.
2. **Pay reimbursement** turns to retrying: "Attempt 1 failed: payment provider unavailable (simulated outage). Temporal will retry it", with a "Next attempt in" countdown. The code panel highlights the `RetryPolicy` lines.
3. After three failures it succeeds: "Payment reference PAY-… · succeeded on attempt 4". The status reads **Paid**.
4. Say: "The provider failed three times. There's no retry loop in this code; the Retry Policy handles it, and the Workflow ID is the payment's idempotency key, so retries can't pay twice."

### 8. Over the policy limit

1. Click **Laptop · $7,200**, then **Submit expense**.
2. The journey shows Check expense policy ("Over the $5,000.00 per-expense limit"), Notify employee, and **Rejected by policy**. No one is asked.

**New session** in the top bar clears your view for the next audience. Each browser has its own private set of requests.

## Show it in code

All links point at commit [`d3418de`](https://github.com/temporal-sa/temporal-demo-expense-approval/commit/d3418de2c7847381d7f794baf8fac834e10c5c08). Open a file locally with `code -g <path>:<line>`.

### Receive the approver's decision as a Signal

**When to use it:** after beat 4, or when someone asks where the decision goes while no worker is running.

**Where it is:** [`expense/workflow.go` lines 38–51](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/workflow.go#L38-L51) (`code -g expense/workflow.go:38`), sent from [`web/server.go` lines 264–270](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/web/server.go#L264-L270) (`code -g web/server.go:264`)

```go
decisions := workflow.GetSignalChannel(ctx, DecisionSignal)
workflow.Go(ctx, func(ctx workflow.Context) {
	for {
		var d Decision
		decisions.Receive(ctx, &d)
		waiting := status.Stage == StageAwaitingApproval || status.Stage == StageEscalated
		allowed := d.By == req.Approver || (status.Stage == StageEscalated && d.By == req.EscalationApprover)
		if !waiting || status.Decision != nil || !allowed {
			logger.Info("decision ignored", "by", d.By, "stage", status.Stage)
			continue
		}
		status.Decision = &d
	}
})
```

```go
// A Signal is accepted by Temporal even when no worker is running. The
// Workflow acts on it as soon as a worker is available.
d := expense.Decision{Approved: in.Approved, By: by, Comment: strings.TrimSpace(in.Comment), DecidedAt: time.Now().UTC()}
if err := s.cfg.Client.SignalWorkflow(ctx, wfID, "", expense.DecisionSignal, d); err != nil {
	writeTemporalError(w, err)
	return
}
```

**What you need to understand:**

- `SignalWorkflow` returns once Temporal has written the Signal into the Workflow's Event History. No worker is involved, which is why **Approve** works in beat 4.
- When a worker next runs the Workflow, the Signal is delivered to the channel and the handler sets `status.Decision`.
- The Workflow enforces the rules itself: it only takes a decision while it's waiting, only from the approver (or from Dana once escalated), and only the first one. A double-click or a late reply is logged and ignored. The web app checks the same rules first so the approver gets a clear error, but it isn't the guard.
- The decision carries who, the verdict, a comment, and when. That makes the Event History the audit trail.
- A Signal doesn't return a result to the sender. If the approver needed an immediate "accepted" or "too late" answer from the Workflow itself, an Update would do that, but it needs a worker running to answer.

**Docs:** [Signals in Go](https://docs.temporal.io/develop/go/message-passing#signals), [Approval pattern](https://docs.temporal.io/design-patterns/approval)

### Wait for a decision or a deadline, and cancel the timer when the decision wins

**When to use it:** at beat 2 when the countdown appears, at beat 5 when the history shows `TimerCanceled`, or at beat 6 when it escalates.

**Where it is:** [`expense/workflow.go` lines 121–129](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/workflow.go#L121-L129) (`code -g expense/workflow.go:121`), called from [lines 71–86](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/workflow.go#L71-L86)

```go
func awaitDecision(ctx workflow.Context, timeout time.Duration, decided func() bool) (bool, error) {
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	defer cancelTimer()
	timer := workflow.NewTimer(timerCtx, timeout)
	if err := workflow.Await(ctx, func() bool { return decided() || timer.IsReady() }); err != nil {
		return false, err
	}
	return decided(), nil
}
```

```go
decided := func() bool { return status.Decision != nil }
ok, err := awaitDecision(ctx, req.EscalateAfter, decided)
// ...
if !ok {
	status.Stage = StageEscalated
	notice := ApprovalNotice{To: req.EscalationApprover, Request: req, Escalated: true}
	if err := workflow.ExecuteActivity(ctx, a.NotifyApprover, notice).Get(ctx, nil); err != nil {
		return ExpenseResult{}, err
	}
	ok, err = awaitDecision(ctx, req.ExpireAfter-req.EscalateAfter, decided)
	// ...
}
```

**What you need to understand:**

- `workflow.NewTimer` is a durable timer: Temporal stores it on the server and fires it on time whether or not a worker is running. The worker doesn't hold anything while the Workflow waits.
- `workflow.Await` unblocks on whichever happens first: the decision Signal or the timer.
- The timer runs on its own cancelable context. When the decision wins, `cancelTimer` runs and the history records `TimerCanceled`, instead of a timer firing later for nothing.
- The two waits are the escalation deadline and then the remaining time to expiry. They come from `ESCALATE_AFTER` (45 seconds) and `EXPIRE_AFTER` (10 minutes), which the web app passes in with each request. In production these would be days; the code doesn't change.

**Docs:** [Durable timers in Go](https://docs.temporal.io/develop/go/timers)

### Retry the payment, and make it safe to retry

**When to use it:** at beat 7, or when someone asks what happens if the payment provider is down for an hour.

**Where it is:** [`expense/workflow.go` lines 16–25](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/workflow.go#L16-L25) (`code -g expense/workflow.go:16`) and [`expense/activities.go` lines 59–75](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/activities.go#L59-L75) (`code -g expense/activities.go:59`)

```go
ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Second,
	// No maximum attempts: a payment provider that is down for an hour
	// should delay the reimbursement, not lose it.
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
	},
})
```

```go
// PayReimbursement pays the employee. The Workflow ID is a stable idempotency
// key: a real payment API that receives the same key on every retry pays at
// most once, so Temporal can retry this step safely.
func (a *Activities) PayReimbursement(ctx context.Context, req ExpenseRequest) (string, error) {
	info := activity.GetInfo(ctx)
	// ...
	// Demo failure injection: fail the first N attempts so the Retry Policy is visible.
	if int(info.Attempt) <= req.Demo.PaymentFailures {
		return "", errors.New("payment provider unavailable (simulated outage)")
	}
	// ...
	activity.GetLogger(ctx).Info("reimbursement paid", /* ... */ "idempotencyKey", info.WorkflowExecution.ID)
	return ref, nil
}
```

**What you need to understand:**

- The Retry Policy waits 1 second after the first failure and doubles the wait each time, up to 30 seconds. With no `MaximumAttempts`, it keeps retrying until the Activity succeeds. In beat 7 the waits are 1, 2 and 4 seconds.
- `StartToCloseTimeout` bounds each attempt to 10 seconds. A hung call is abandoned and retried.
- The policy applies to every Activity in the Workflow, not only the payment.
- Retries mean an attempt can run more than once, for example if the worker dies after the provider paid but before it reported success. The Workflow ID is passed as the idempotency key, so a real provider that honors it pays at most once. The demo logs the key; it has no real provider.
- A customer would mark errors that will never succeed (an invalid account, say) as non-retryable, so the Workflow stops retrying them. This demo has none.

**Docs:** [Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies), [Activity idempotency](https://docs.temporal.io/activity-definition#idempotency)

### Show progress without a worker

**When to use it:** at beat 4, when someone asks how the screen still updates while the worker is down.

**Where it is:** [`web/history.go` lines 15–18](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/web/history.go#L15-L18) (`code -g web/history.go:15`) and [lines 293–302](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/web/history.go#L293-L302)

```go
// The UI is built from the Workflow's Event History and DescribeWorkflowExecution
// rather than a Query. Both are served by Temporal itself, so every screen keeps
// working while no worker is running, which is exactly when the demo is most
// interesting.
```

```go
// A workflow task first goes to the last worker's sticky queue and is
// rescheduled if that worker is gone, so its scheduled time can reset.
// The first unprocessed event says how long the work has really been ready.
if wt := desc.GetPendingWorkflowTask(); wt != nil && wt.GetState() == enumspb.PENDING_WORKFLOW_TASK_STATE_SCHEDULED {
	ready := wt.GetScheduledTime().AsTime()
	if unprocessedSince != nil && unprocessedSince.Before(ready) {
		ready = *unprocessedSince
	}
	noteWaiting(ready)
}
```

**What you need to understand:**

- The Workflow does register a `status` Query, but a Query is answered by a worker running the Workflow code. With the worker stopped, a Query can't answer, so the UI doesn't use it.
- Event History and Describe come from the Temporal Service, so they work with no worker at all. The UI turns events into journey steps, and a pending Task that has waited more than 3 seconds into the **No worker is running this step.** banner.
- A `WorkflowTaskTimedOut` in beat 5's history comes from this sticky queue: the Task first went to the stopped worker, timed out, and was rescheduled on the shared Task Queue.
- This view is demo-specific. A real app would usually show its own status, from a Query when workers are healthy or from its own database.

### What's demo-only

When someone asks what they'd change for their own system, these are the parts to leave out:

- **The Activities only simulate their calls.** [`expense/activities.go` lines 18–21](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/activities.go#L18-L21): each step sleeps under a second and returns a message. A real struct would hold clients for the policy service, email and payments.
- **Failure injection lives in the request, read only by the Activity.** [`expense/types.go` lines 34–40](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/types.go#L34-L40): `PaymentFailures` makes the first attempts fail. The Workflow never reads it, so the Workflow code is what a customer would ship.
- **Timers are seconds, not days.** [`expense/types.go` lines 25–29](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/expense/types.go#L25-L29): the demo passes 45 seconds and 10 minutes.
- **One browser plays every role, and each browser gets its own session.** [`web/server.go` lines 330–335](https://github.com/temporal-sa/temporal-demo-expense-approval/blob/d3418de2c7847381d7f794baf8fac834e10c5c08/web/server.go#L330-L335): Workflow IDs are `<task queue>-<session>-<expense id>`, so presenters and crashable environments sharing a Namespace never see each other's requests. A real app would use its own expense ID.
- **The journey view is built from raw history** (previous entry). That's a demo choice to keep the screen alive during the outage.

## Reset and tear down

- **Between runs:** click **New session**. Your view starts empty. Earlier requests keep running in Temporal until they finish or expire, at most 10 minutes later.
- **Before you leave a crashable environment:** if you scaled workers down, click **Scale up workers** so open requests can finish. Then click **Delete**, or let it delete itself at the end of the hour.
- **Locally:** stop `make web`, `make worker` and `make temporal` with `Ctrl-C`. The dev server keeps nothing.
- Nothing in this demo runs on a schedule. A request that nobody decides costs a timer and an expiry, then completes. Each session can have at most 10 open requests.

## When something goes wrong

| What you see | Likely cause | Fix |
|---|---|---|
| "Reconnecting to Temporal…" under **My requests** | The web app can't reach Temporal | Locally, start `make temporal`. On tmprl-demo.cloud, check `/healthz` on the app's URL |
| A new request shows only "Submitted by Alex Rivera" and the status **Waiting for a worker** | No worker is polling the Task Queue | Click **Scale up workers**, or start `make worker` |
| The request escalated before you could approve | The 45-second countdown ran out | Use it as beat 6, then submit another for beats 2 to 5. Locally, set a longer `ESCALATE_AFTER` before `make web` |
| A toast says "This request isn't waiting for Sam Chen" | The request already has a decision, or it expired or finished | Pick a request that's in the inbox |
| "You already have 10 open requests. Decide on some first" | The per-session limit | Decide on some, or click **New session** |
| After **Crash workers**, the outage is over in seconds | Kubernetes replaced the pods | Use **Scale down workers** for a longer outage |
| **Open execution ↗** asks you to sign in | Temporal Cloud needs your own login for the Namespace | Sign in, or show the journey panel instead |

## Keeping this runbook current

The code links point at a fixed commit. When a linked file changes, update its link, line numbers, and snippet together. Before an event, list the linked files that have changed:

```sh
grep -oE 'blob/[0-9a-f]{40}/[^#)]+' RUNBOOK.md | sort -u | while IFS=/ read -r _ sha file; do
  git diff --quiet "$sha" HEAD -- "$file" || echo "changed since ${sha:0:7}: $file"
done
```
