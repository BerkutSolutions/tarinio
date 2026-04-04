# ADR-003: Config Rollout and Rollback Flow

Status: Accepted
Date: 2026-03-31

## Context

The standalone WAF MVP uses revision-based compiled bundles.
Those bundles must be applied safely on a single node without:
- partial live mutation of active config
- hidden runtime-only state changes
- unsafe recovery after failed activation

The system therefore needs one explicit operational contract for:
- render
- validate
- stage
- activate
- reload
- health-check
- success or failure decision

## Decision

In MVP, every runtime-affecting change is applied through a single-node
revision rollout flow owned by the control plane.

The control plane is responsible for:
- creating the target `Revision`
- validating it before activation
- staging it as a complete candidate bundle
- activating it atomically
- reloading runtime
- evaluating health
- deciding success or failure
- rolling back to the last known good `Revision` when needed

No direct runtime mutation outside this flow is allowed.

## Operational Flow

The rollout flow for one target `Revision` is:

1. `render`
2. `validate`
3. `stage`
4. `activate`
5. `reload`
6. `health-check`
7. `success/failure decision`

## Step 1: Render

The control plane compiler renders a complete revision bundle from the logical model.

Synchronous:
- compiler render
- checksum generation
- bundle manifest generation
- initial `Revision` record creation

Fail conditions:
- compiler error
- missing required logical input
- invalid template rendering
- manifest/checksum generation failure

Recorded state:
- `Revision.status = created` on successful render
- `Revision.status = failed` if render fails
- `Job.status = failed` if render fails inside an apply job
- `Event` with type `config.render.succeeded` or `config.render.failed`
- `AuditEvent` for operator-triggered apply attempts

## Step 2: Validate

The control plane validates the rendered candidate bundle before it can be activated.

Validation includes MVP checks such as:
- required artifact presence
- bundle integrity and manifest consistency
- runtime config syntax validation for generated NGINX config
- runtime config syntax validation for generated ModSecurity-related includes where applicable

Synchronous:
- pre-activation validation runs synchronously within the apply operation

Fail conditions:
- missing artifact
- checksum mismatch
- invalid bundle structure
- NGINX config test failure
- ModSecurity-related config include failure

Recorded state:
- `Revision.status = validated` on successful validation
- `Revision.status = failed` on validation failure
- `Job.status = failed` on validation failure
- `Event` with type `config.validate.succeeded` or `config.validate.failed`
- `AuditEvent` for failed validation if operator-triggered

Rollback behavior:
- no rollback is needed because activation has not happened yet
- current active revision remains unchanged

## Step 3: Stage

The validated bundle is copied or prepared in a candidate runtime location separate from the currently active revision.

Synchronous:
- candidate directory/materialization
- file placement for the target revision

Fail conditions:
- candidate path preparation failure
- file copy/write failure
- incomplete staged bundle
- staged bundle checksum mismatch

Recorded state:
- `Event` with type `config.stage.succeeded` or `config.stage.failed`
- `Revision.status` remains `validated` if stage succeeds
- `Revision.status = failed` if stage fails
- `Job.status = failed` if stage fails

Rollback behavior:
- no runtime rollback is needed because active revision is still unchanged

## Step 4: Activate

Activation switches the runtime's active reference from the current revision to the staged target revision.

For MVP, activation must be atomic at the revision boundary.
Allowed implementations include:
- switching an active symlink
- switching an active directory pointer
- replacing one active revision reference with another in one controlled step

Synchronous:
- active reference switch

Fail conditions:
- active reference switch failure
- target staged bundle not present at activation time
- target revision identity mismatch

Recorded state:
- `Event` with type `config.activate.succeeded` or `config.activate.failed`
- `Revision.status` stays non-active until reload and health-check complete
- `Revision.status = failed` if activation step fails
- `Job.status = failed` if activation step fails

Rollback behavior:
- if activation fails before runtime reload, restore previous active reference immediately

## Step 5: Reload

After activation, the control plane triggers runtime reload against the newly active revision.

Synchronous:
- reload command execution
- immediate reload exit/success capture

Fail conditions:
- reload command returns error
- runtime rejects the active config during reload
- runtime process becomes unavailable immediately after reload

Recorded state:
- `Event` with type `config.reload.succeeded` or `config.reload.failed`
- `Revision.status = failed` on reload failure
- `Job.status = failed` on reload failure

Rollback behavior:
- rollback is triggered immediately on reload failure

## Step 6: Health-Check

After a successful reload, the control plane performs post-apply health checks for the active revision.

For MVP single-node scope, health-check means:
- runtime process is alive
- health endpoint or equivalent runtime readiness check succeeds
- active revision identity matches the intended revision

Synchronous:
- immediate post-reload health evaluation

May also be represented by a short-lived apply `Job`, but the decision window remains part of the same apply operation.

Fail conditions:
- runtime health endpoint fails
- runtime does not become ready in the allowed window
- active revision does not match the intended target

Recorded state:
- `Event` with type `config.health_check.succeeded` or `config.health_check.failed`
- `Revision.status = active` only after successful health-check
- `Job.status = succeeded` only after successful health-check

Rollback behavior:
- rollback is triggered immediately on health-check failure

## Step 7: Success / Failure Decision

The control plane decides rollout result only after health-check completes.

Success means:
- render succeeded
- validate succeeded
- stage succeeded
- activate succeeded
- reload succeeded
- health-check succeeded

On success, recorded state is:
- `Revision.status = active`
- previous active revision remains available as rollback candidate
- `Job.status = succeeded`
- `Event` with type `config.apply.succeeded`
- `AuditEvent` with success status for operator-triggered apply

Failure means any step above failed.

On failure, recorded state is:
- target `Revision.status = failed` unless rollback later marks it differently by policy
- `Job.status = failed`
- `Event` with type `config.apply.failed`
- `AuditEvent` with failure status for operator-triggered apply

## Rollback Trigger Conditions

Rollback is triggered when:
- activation fails after the active reference has been touched
- reload fails
- post-reload health-check fails
- active revision identity cannot be confirmed after activation

Rollback is not required when failure happens before activation and the old revision is still untouched.

## Last Known Good Revision

The `last known good Revision` is the most recent revision that:
- completed validation
- was activated
- reloaded successfully
- passed post-apply health-check
- has `Revision.status = active` or an equivalent previously successful status in history

For single-node MVP, the control plane must persist enough revision metadata to identify:
- current active revision
- immediately previous successful revision
- historical successful revisions eligible for manual recovery if needed

Automatic rollback should select the most recent successful revision before the failed target revision.

## Safe Return Guarantee

Safe rollback in MVP is guaranteed by these rules:
- rollback target must already be a previously validated successful revision
- rollback uses full revision activation, not file-by-file repair
- rollback follows the same activate -> reload -> health-check pattern
- failed candidate revision must not remain the active revision after rollback succeeds

If rollback itself fails:
- system records explicit critical failure events
- current runtime state is marked degraded
- no silent success is allowed

## Idempotency

Apply must be idempotent at the revision level.

MVP rules:
- re-applying the already active revision must not corrupt runtime state
- repeated apply of the same validated revision must converge to the same active runtime result
- duplicate operator requests targeting the same revision must not create inconsistent active references

Recorded state for idempotent re-apply may include:
- `Event` such as `config.apply.noop` or `config.apply.reconfirmed`
- `Job.status = succeeded` if runtime already matches the requested revision and health is good

## Failure Isolation

Failure isolation in single-node MVP means:
- one invalid site definition must fail candidate revision validation before activation
- invalid config for one site must not partially mutate the active runtime
- failure of one candidate revision must not remove or corrupt the currently active good revision

Because runtime is single-node and bundle-based in MVP:
- active runtime remains on the last known good full revision if the candidate fails
- the system does not attempt per-site live patching inside the active bundle

## Sync vs Job Execution

Synchronous inside the apply operation:
- render
- validate
- stage
- activate
- reload
- immediate health-check
- success/failure decision

Tracked through `Job`:
- the overall apply request and its lifecycle
- retries initiated by the control plane
- operator-visible status history for apply attempts

In MVP, apply is operationally synchronous but still recorded as a `Job` for traceability.

## Single-Node Scope

This ADR applies only to single-node MVP deployment.
It does not define:
- distributed rollout coordination
- multi-runtime consistency
- fleet orchestration
- quorum or leader election behavior

## Resulting Rule

No runtime-affecting change is complete unless it:
- produces a `Revision`
- passes render, validate, and stage
- is activated atomically
- reloads successfully
- passes health-check
- can roll back safely to the last known good revision


