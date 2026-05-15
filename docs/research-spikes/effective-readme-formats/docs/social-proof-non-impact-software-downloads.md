# The (Non)-Impact of Social Proof on Software Downloads

- **Source URL**: https://arxiv.org/pdf/2603.07919
- **Retrieved**: 2026-05-15
- **Authors**: Lucas Shen and Gaurav Sood

---

## Overview

This research paper investigates whether social proof mechanisms — specifically GitHub stars — meaningfully influence software download behavior. The study challenges assumptions about visibility signals in open-source ecosystems.

## Research Methodology

The authors conducted experiments examining GitHub repositories to understand whether star counts function as effective social proof. Their approach involved:

- **GitHub Repository Analysis**: Systematic examination of repository characteristics and engagement metrics
- **Download Data**: Collection of PyPI (Python Package Index) download statistics to measure actual adoption
- **Experimental Design**: Tests isolating the effect of visible social proof signals from other confounding factors

The research included balance tests across repository characteristics and analysis of heterogeneous effects by package complexity.

## Key Findings

The central discovery contradicts common assumptions: social proof displays demonstrate limited persuasive power in driving software adoption decisions. Specifically:

- Star counts showed minimal correlation with download volumes when controlling for package utility
- PyPI download metrics remained largely unaffected by GitHub visibility signals
- The effect persisted even when examining packages of varying complexity levels
- Observational data suggested social proof mechanisms operate differently in developer communities than in consumer contexts

## Why Developers May Be Less Susceptible to Social Proof

Developers face tangible consequences from poor choices (broken builds, security vulnerabilities, maintenance burden), giving them strong motivation to evaluate quality through the central route. They can also draw on richer signals than raw popularity, including code documentation, commit frequency, contributor activity, and project responsiveness.

## Conclusions

The authors conclude that while visibility matters in software ecosystems, traditional social proof metrics like stars function inadequately as adoption signals. Developer decision-making appears driven primarily by functional requirements and community reputation mechanisms beyond simple star counts.

## Implications

GitHub stars serve community recognition purposes rather than persuasion functions, potentially reshaping how platforms should design incentive structures for open-source projects. Bad actors can also game social proof metrics to induce the use of malign software, raising concerns about the reliability of star ratings as a selection criterion.
