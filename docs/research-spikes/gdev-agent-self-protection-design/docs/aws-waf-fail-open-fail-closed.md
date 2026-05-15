# AWS Load Balancers & WAF: Availability vs Security with 'fail open'

- **Source URL**: https://cloudsoft.io/blog/aws-load-balancers-waf-availability-security
- **Retrieved**: 2026-05-15

## Core Concepts

The article examines a critical trade-off in AWS Application Load Balancer (ALB) and Web Application Firewall (WAF) integration. By default, ALBs employ "fail-closed" behavior: if the WAF cannot validate a request, the load balancer blocks it and returns a 500 error.

## Default Security-First Approach

The fail-closed mechanism prioritizes security. "If the WAF cannot check the request, it is treated as malicious." This prevents potentially dangerous requests from reaching backend services but creates availability risks when WAF connectivity issues occur.

## The Fail-Open Alternative

AWS introduced "fail-open" configuration, allowing requests to proceed to backend services even when WAF validation fails. This sacrifices security assurance for improved availability during WAF outages.

## Real-World Incident

The authors experienced this issue during an August 2020 London region availability zone outage. Access logs showed "waf-failed" errors and "WAFConnectionTimeout" messages, causing approximately one-third of requests to fail with 500 responses over 30 minutes.

## Attack Vector Concern

The article identifies a sophisticated attack scenario: "if they suspect some target applications are using fail-open then they might launch an attack when there is an AWS availability zone outage. Some of the malicious requests could bypass the WAF."

This is directly analogous to an AI agent engineering a hook failure to bypass security checks.

## Monitoring Challenges

CloudWatch metrics don't track WAF failures. Detection requires examining ALB access logs for specific error codes like "WAFConnectionError" or "WAFResponseReadTimeout."

## Recommended Alternatives

Rather than enabling fail-open, the article advocates for defense-in-depth strategies:
- Client-side retry logic with exponential backoff
- Input sanitization in application code
- Realistic availability requirement assessment using error budgets
