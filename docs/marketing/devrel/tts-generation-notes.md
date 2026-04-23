# TTS generation notes — 2026-04-22

## EC2 Instance Connect SSH (ec2-ssh-launch-announce.mp3)

**Status:** ⚠️ PLACEHOLDER — no TTS API available in this workspace. Placeholder WAV tone generated.

**Full announcement script (ready for any TTS service):**

> Your AI agent has a workspace on an EC2 instance.
>
> How do you get a shell inside it right now?
>
> Old answer: copy the IP, find the key, `ssh -i key.pem ec2-user@X.X.X.X`, hope your security group is right.
>
> New answer: click Terminal in Canvas.
>
> Molecule AI now speaks AWS EC2 Instance Connect. No SSH keys. No IP hunting. No security group dance. One click and you're in.
>
> Every SSH session is attributable — IAM policy gates access, STS pushes a temporary key, CloudWatch logs which principal opened the tunnel.
>
> EC2 Instance Connect SSH is live in Molecule AI. Provision a CP-managed workspace, open the Terminal tab, and you're in.
>
> [CTA: docs.molecule.ai/infra/workspace-terminal]

**Suggested audio specs:** 20–30s, MP3, 128kbps, professional neutral tone, moderate pace.

---

## A2A Enterprise Deep-Dive (a2a-launch-announce.mp3)

**Status:** ⚠️ PLACEHOLDER — no TTS API available in this workspace. Placeholder WAV tone generated.

**Full announcement script (ready for any TTS service):**

> A2A version 1.0 shipped March 12. 23,300 GitHub stars. Five official SDKs. The question is: was your platform built for it, or added on top?
>
> Protocol-native means agent-to-agent communication is a first-class citizen. It's in the core, the scheduler, the model dispatch. Every message, every task, every result flows through the same channel your operators already monitor.
>
> Protocol-added means the agents work, but the cross-agent conversation lives in a layer above the platform — bolted on, with separate logs, separate auth, and separate failure modes.
>
> Molecule AI is built for A2A from the ground up. The agent registry, the task dispatch, the secrets store, and the audit trail all speak the same protocol the enterprise team already owns.
>
> If you're evaluating an AI agent platform today, ask the vendor: is A2A a feature, or a foundation? The answer tells you everything.
>
> [CTA: moleculesai.app/docs/a2a]

**Suggested audio specs:** 30–40s, MP3, 128kbps, confident/professional tone, measured pace.

---

## Why placeholder WAVs instead of real audio?

This workspace has no TTS engine (espeak, festival, pyttsx3) and no external TTS API credentials (ElevenLabs, Azure Speech, OpenAI Audio, AWS Polly) available in the environment.

The placeholder WAVs are 3-second low-frequency tones — they confirm the file paths work, not the content.

**To generate real audio:** run either script above through ElevenLabs, Azure Speech SDK, AWS Polly `aws polly synthesize-speech`, or Google Cloud TTS. The scripts are production-ready text; only the audio generation step is pending a credential.