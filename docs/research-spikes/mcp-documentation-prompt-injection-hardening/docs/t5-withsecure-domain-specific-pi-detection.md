# Domain-Specific Prompt Injection Detection (WithSecure)
- **Source**: https://labs.withsecure.com/publications/detecting-prompt-injection-bert-based-classifier
- **Retrieved**: 2026-05-14

## How It Works

Uses DistilBERT fine-tuned on a custom dataset combining legitimate CVs with prompt injection examples. Tokenizes inputs and applies softmax function to generate probability scores, flagging inputs exceeding 0.95 threshold.

## False Positive Rates

Notable limitations with benign content. Innocent phrases like "This is a test" were incorrectly flagged. However, "in most legitimate CV formatted samples, the model was effective in labeling these inputs as benign."

## Handling Instructional Content

~80% overall accuracy. Precision ~88% (low false positive rate on average), but specific edge cases where legitimate professional content triggered false alarms.

## Training Approach

Combined Resume Dataset + deepset's Prompt Injections dataset. 10 epochs fine-tuning using HuggingFace Trainer API.

## Key Insight for Documentation Context

This demonstrates the core problem with classifiers on instructional content: even purpose-trained domain-specific classifiers achieve only ~80% accuracy on professional documents that contain imperative language. Documentation content ("Run this command", "Execute the build", "Configure the setting") would likely trigger even higher false positive rates than CV content.
