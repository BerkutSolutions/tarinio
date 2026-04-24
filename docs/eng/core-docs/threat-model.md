# Threat Model

This page belongs to the current documentation branch.

## Purpose

This document explains what TARINIO is designed to help defend, what boundaries exist, and where operator responsibility remains.

## Protected Assets

TARINIO primarily protects:

- application ingress;
- policy-controlled request handling;
- controlled release of traffic protection settings;
- operator visibility into security and rollout events.

## Main Trust Boundaries

Important boundaries are:

- public client -> runtime ingress;
- runtime -> upstream applications;
- operator -> control-plane;
- control-plane -> PostgreSQL / Redis / runtime API.

## Threat Classes TARINIO Helps Address

- common web attack traffic inspected through WAF/CRS;
- abusive request patterns and brute-force-like traffic;
- L4/L7 hostile traffic patterns controlled by Anti-DDoS features;
- unsafe configuration drift through revision-based change flow.

## Shared Responsibility

TARINIO does not remove the need for:

- secure application code;
- network segmentation;
- secret management;
- backup and disaster recovery;
- host and container hardening.

## Non-Goals

TARINIO is not, by itself:

- a SIEM;
- a full IAM or directory service;
- an EDR product;
- a replacement for secure application development.
