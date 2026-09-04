                    THEMIS
        Security Application / Authority
┌───────────────────────────────────────────────┐
│                                               │
│  Security Governance                          │
│  ├── Findings                                 │
│  ├── Enterprise Positions                     │
│  ├── Security decisions                       │
│  └── Security workflows                       │
│                                               │
│  Knowledge Builder                            │
│  ├── CVE/CWE/CPE                              │
│  ├── Vendor/Product knowledge                 │
│  └── Enterprise security knowledge             │
│                                               │
│  SBOM / Scan / Release / Product data         │
│                                               │
│  ┌─────────────────────────────────────────┐  │
│  │       AI Harness / Agent Runtime        │  │
│  │                                         │  │
│  │  1 Instructions                         │  │
│  │  2 Context Delivery                     │  │
│  │  3 Context Management                   │  │
│  │  4 Tool Interface                       │  │
│  │  5 Execution Environment                │  │
│  │  6 Durable State                        │  │
│  │  7 Orchestration                        │  │
│  │  8 Subagents                            │  │
│  │  9 Skills                               │  │
│  │ 10 Verification / Observability         │  │
│  │ 11 Ratchet                              │  │
│  │                                         │  │
│  │             Model Adapter               │  │
│  └──────────────────┬──────────────────────┘  │
│                     │                         │
└─────────────────────┼─────────────────────────┘
                      ▼
                 ┌─────────┐
                 │ DeepSeek │
                 └─────────┘

The critical point is that DeepSeek should not know that it is the security system. It reasons over context supplied by Themis/Harness.
