<!-- Source: https://docs.aws.amazon.com/waf/latest/developerguide/web-acl-testing.html -->
<!-- Retrieved: 2026-05-15 -->

# Testing and Tuning AWS WAF Protections

## Core Guidance

AWS recommends testing and tuning any changes to your AWS WAF protection pack (web ACL) before applying them to your website or web application traffic.

**Production traffic risk**: Before deploying your protection pack implementation for production traffic, test and tune it in a staging or testing environment until you are comfortable with the potential impact to your traffic. Then test and tune the rules in count mode with your production traffic before enabling them.

## Count Mode Implementation

When a rule has a Count action, AWS WAF counts the request but does not determine whether to allow or block it. The request continues processing through remaining rules in the web ACL.

### Key Behaviors

- Count is a non-terminating action
- Counted requests are listed under `nonTerminatingMatchingRules` in WAF logs
- Count action rules in rule groups do NOT emit web ACL dimension metrics — only Rule, RuleGroup, and Region dimensions

### Deployment Strategy

1. Deploy in Count mode first by setting OverrideAction to Count for new rule groups
2. Monitor for 1-2 weeks to review sampled requests and check for false positives
3. Exclude problematic rules using rule overrides
4. Switch to Block mode

## Monitoring

Using CloudWatch metrics for AWS WAF, you can:
- Create dashboard graphs showing allowed, blocked, or counted requests over time per rule
- See metrics by label
- Analyze logs using CloudWatch Logs Insights

## Temporary Inconsistencies During Updates

Changes take time to propagate (seconds to minutes). During propagation:
- New rule group rules might be in effect in one area but not another
- Changed rule action settings might show old actions in some places and new in others
- New IP set entries might be blocked in one area while still allowed in another
