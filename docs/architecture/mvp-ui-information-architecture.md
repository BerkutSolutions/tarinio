# MVP UI Information Architecture

Status: Finalized for Stage 0 task `Freeze MVP UI information architecture`
Date: 2026-03-31

## Purpose

This document fixes the MVP UI information architecture for the standalone WAF.

It defines:
- top-level navigation
- main product sections
- key screens
- main user flows

It does not define:
- visual design
- layout styling
- component library details
- enterprise or plugin navigation

## Core Rule

The UI is a control-plane client.

The UI:
- does not invent entities
- does not define backend behavior
- uses the already defined domain model
- reads and mutates data only through control-plane APIs

## Top-Level Navigation

The MVP top-level navigation is:
- `Dashboard`
- `Sites`
- `Policies`
- `TLS & Certificates`
- `Events`
- `Access Control`
- `Jobs`
- `Administration`

This navigation is fixed for MVP.
No plugin pages, Pro sections, enterprise sections, or fleet-management sections are included.

## Section: Dashboard

## Purpose

`Dashboard` gives the operator a current operational summary.

## Main Screens

- dashboard overview

## Data Shown

The section shows:
- requests count summary
- blocked requests summary
- top IPs summary
- top rules summary
- certificate status summary
- latest apply/revision status
- recent important events

API data shape:
- report summary endpoints
- recent events endpoint
- recent revision/apply status endpoint

## Available Actions

- open site details
- open recent events
- open revision/apply status
- open certificate status details

## Entities Used

- `Event`
- `Revision`
- `Job`
- `Certificate`

## Section: Sites

## Purpose

`Sites` is the main place to manage protected applications.

## Main Screens

- site list
- site details
- create site
- edit site

## Data Shown

The section shows:
- list of `Site`
- linked default `Upstream`
- linked `TLSConfig`
- linked `WAFPolicy`
- linked `AccessPolicy`
- linked `RateLimitPolicy`
- last apply/revision status for the site

API data shape:
- site list endpoint
- site details endpoint
- upstream details for selected site
- latest revision/apply status for selected site

## Available Actions

- create site
- edit site
- enable or disable site
- set default upstream
- navigate to related policy, TLS, access, and job views

## Entities Used

- `Site`
- `Upstream`
- `TLSConfig`
- `WAFPolicy`
- `AccessPolicy`
- `RateLimitPolicy`
- `Revision`

## Section: Policies

## Purpose

`Policies` manages request inspection and traffic-control behavior attached to sites.

## Main Screens

- policy list by site
- site policy details
- edit WAF policy
- edit rate-limit policy

## Data Shown

The section shows:
- `WAFPolicy` status and mode
- CRS enablement
- custom rule presence
- `RateLimitPolicy` settings
- related site linkage

API data shape:
- policy read endpoints
- site-linked policy endpoints

## Available Actions

- enable or disable WAF
- switch detection or blocking mode
- enable or disable CRS
- update custom rule text within MVP limits
- enable or disable rate limit
- update rate-limit thresholds

## Entities Used

- `Site`
- `WAFPolicy`
- `RateLimitPolicy`

## Section: TLS & Certificates

## Purpose

`TLS & Certificates` manages site TLS configuration and certificate lifecycle state.

## Main Screens

- certificate list
- certificate details
- site TLS configuration
- certificate assignment
- issuance/renewal status view

## Data Shown

The section shows:
- `TLSConfig`
- `Certificate`
- certificate status
- expiry windows
- last renewal or issuance result
- site-to-certificate assignment

API data shape:
- certificate list endpoint
- certificate details endpoint
- site TLS endpoint
- renewal/issuance job status endpoint

## Available Actions

- assign certificate to site
- enable site TLS
- enable HTTP to HTTPS redirect
- upload certificate manually
- trigger issuance or renewal flow when supported

## Entities Used

- `Site`
- `TLSConfig`
- `Certificate`
- `Job`
- `Event`

## Section: Events

## Purpose

`Events` is the operational event feed for recent important system and security activity.

## Main Screens

- event list
- filtered event view
- event details

## Data Shown

The section shows:
- normalized security `Event`
- apply failures
- certificate failures
- rate-limit hits
- access policy denies

API data shape:
- event list endpoint
- event detail endpoint
- filter parameters for recent operational use

## Available Actions

- filter by site
- filter by type
- filter by severity
- open related site, job, or revision context

## Entities Used

- `Event`
- `Site`
- `Job`
- `Revision`
- `Certificate`

## Section: Access Control

## Purpose

`Access Control` manages site access restrictions for MVP.

## Main Screens

- access policy list by site
- access policy editor
- manual ban/unban view

## Data Shown

The section shows:
- `AccessPolicy`
- allow and deny CIDRs
- trusted proxy CIDRs
- manual ban state when supported by MVP implementation

API data shape:
- access policy endpoints
- manual ban/unban endpoints

## Available Actions

- edit allow CIDRs
- edit deny CIDRs
- edit trusted proxy CIDRs
- ban IP manually
- unban IP manually

## Entities Used

- `Site`
- `AccessPolicy`
- related security `Event`

## Section: Jobs

## Purpose

`Jobs` gives operators visibility into background execution and rollout activity.

## Main Screens

- jobs list
- job details
- recent apply jobs
- recent certificate jobs

## Data Shown

The section shows:
- `Job` type and status
- linked `Revision`
- linked `Certificate`
- linked `Site`
- timestamps
- result summaries

API data shape:
- jobs list endpoint
- job details endpoint

## Available Actions

- inspect job result
- inspect linked revision
- inspect linked certificate failure
- inspect linked site or event history

## Entities Used

- `Job`
- `Revision`
- `Certificate`
- `Site`
- `Event`

## Section: Administration

## Purpose

`Administration` manages local administrative access and accountability.

## Main Screens

- user list
- role list
- audit log
- user details
- role details

## Data Shown

The section shows:
- `User`
- `Role`
- permissions
- `AuditEvent`
- recent critical admin changes

API data shape:
- user endpoints
- role endpoints
- audit endpoints

## Available Actions

- create or edit user
- activate or deactivate user
- assign roles
- inspect audit history

## Entities Used

- `User`
- `Role`
- `AuditEvent`

## Key User Flows

## Flow: Create Site

1. operator opens `Sites`
2. operator selects `create site`
3. UI sends site creation data to site API
4. UI links or creates default upstream settings
5. UI returns to site details with related sections available

Entities:
- `Site`
- `Upstream`

Required API areas:
- sites create/read/update
- upstream create/read/update

## Flow: Enable TLS

1. operator opens a site or `TLS & Certificates`
2. operator configures `TLSConfig`
3. operator uploads or assigns `Certificate`
4. UI saves TLS state through API
5. UI shows resulting certificate and revision/apply status

Entities:
- `Site`
- `TLSConfig`
- `Certificate`
- `Revision`
- `Job`

Required API areas:
- site TLS endpoints
- certificate endpoints
- revision/apply status endpoints

## Flow: Enable WAF

1. operator opens `Policies`
2. operator selects the site's `WAFPolicy`
3. operator enables WAF and sets mode and CRS options
4. UI saves WAF policy through API
5. UI shows resulting apply or revision status

Entities:
- `Site`
- `WAFPolicy`
- `Revision`
- `Job`

Required API areas:
- WAF policy endpoints
- revision/apply status endpoints

## Flow: View Blocks

1. operator opens `Events`
2. operator filters for blocked security activity
3. UI shows recent normalized security events
4. operator opens related site or rule context

Entities:
- `Event`
- `Site`

Required API areas:
- event list/filter endpoints

## Flow: Ban IP

1. operator opens `Access Control`
2. operator enters an IP or CIDR to deny
3. UI saves the access change through API
4. UI shows updated access policy and related recent events

Entities:
- `AccessPolicy`
- `Site`
- `Event`

Required API areas:
- access policy endpoints
- manual ban/unban endpoints

## Flow: View Certificate Status

1. operator opens `TLS & Certificates`
2. UI loads certificate list and status summary
3. operator opens a certificate or site TLS detail
4. UI shows expiry, issuance, renewal, and failure state

Entities:
- `Certificate`
- `TLSConfig`
- `Site`
- `Job`
- `Event`

Required API areas:
- certificate endpoints
- TLS endpoints
- job and event endpoints for failures

## Flow: View Errors and Events

1. operator opens `Events` or `Jobs`
2. UI loads recent failures and recent operational events
3. operator opens the detailed event, job, site, or revision context
4. UI provides the related operational trail

Entities:
- `Event`
- `Job`
- `Revision`
- `Site`
- `Certificate`

Required API areas:
- events endpoints
- jobs endpoints
- revision status endpoints

## UI to Domain Model Mapping

The UI uses only already defined control-plane entities:
- `Site`
- `Upstream`
- `TLSConfig`
- `Certificate`
- `WAFPolicy`
- `AccessPolicy`
- `RateLimitPolicy`
- `Event`
- `Job`
- `Revision`
- `User`
- `Role`
- `AuditEvent`

No UI section may introduce a domain concept that is not backed by the control-plane model.

## UI to API Mapping

The UI depends on these API areas:
- site management
- upstream management
- WAF policy management
- rate-limit policy management
- access policy and manual ban management
- TLS and certificate management
- event retrieval
- job retrieval
- revision/apply status retrieval
- user, role, and audit administration
- report summary retrieval for dashboard

## Out of Scope for MVP UI IA

The MVP information architecture does not include:
- plugin management pages
- enterprise fleet pages
- SCC integration pages
- advanced reporting builders
- SIEM investigation workspace
- raw runtime log explorer

## Resulting Rule

The MVP UI structure is fixed around a small set of operator tasks:
- configure sites
- configure protection
- manage TLS
- inspect events
- manage access restrictions
- inspect jobs and revisions
- manage local administration

The UI remains a direct reflection of the accepted domain model and control-plane API.


