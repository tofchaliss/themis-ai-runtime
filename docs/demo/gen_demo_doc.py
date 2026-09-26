#!/usr/bin/env python3
"""Generate docs/demo/themis-demo-architecture.html (inline SVG diagrams); print to PDF with headless Chrome — see docs/demo/README.md."""
import html, sys, os

OUT = sys.argv[1]

# ---------- tiny SVG toolkit ----------
class SVG:
    def __init__(self, w, h, title=""):
        self.w, self.h, self.parts, self.title = w, h, [], title
    def box(self, x, y, w, h, lines, fill="#ffffff", stroke="#1f2937", bold_first=True, fs=12, dash=False, rx=6):
        d = ' stroke-dasharray="6,4"' if dash else ""
        self.parts.append(f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{rx}" fill="{fill}" stroke="{stroke}" stroke-width="1.4"{d}/>')
        if isinstance(lines, str): lines = [lines]
        n = len(lines); lh = fs + 4
        y0 = y + h/2 - (n-1)*lh/2 + fs/3
        for i, t in enumerate(lines):
            fw = ' font-weight="600"' if (i == 0 and bold_first) else ""
            self.parts.append(f'<text x="{x+w/2}" y="{y0+i*lh:.1f}" text-anchor="middle" font-size="{fs}"{fw}>{html.escape(t)}</text>')
    def text(self, x, y, t, fs=12, anchor="start", bold=False, fill="#111827", italic=False):
        fw = ' font-weight="600"' if bold else ""
        fi = ' font-style="italic"' if italic else ""
        self.parts.append(f'<text x="{x}" y="{y}" text-anchor="{anchor}" font-size="{fs}" fill="{fill}"{fw}{fi}>{html.escape(t)}</text>')
    def arrow(self, x1, y1, x2, y2, label="", dash=False, color="#1f2937", lx=None, ly=None, fs=11):
        d = ' stroke-dasharray="5,4"' if dash else ""
        self.parts.append(f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="1.4" marker-end="url(#ah)"{d}/>')
        if label:
            mx = lx if lx is not None else (x1+x2)/2
            my = ly if ly is not None else (y1+y2)/2 - 5
            self.parts.append(f'<text x="{mx}" y="{my}" text-anchor="middle" font-size="{fs}" fill="{color}">{html.escape(label)}</text>')
    def path(self, pts, label="", dash=False, color="#1f2937", lx=None, ly=None, fs=11):
        d = ' stroke-dasharray="5,4"' if dash else ""
        p = "M " + " L ".join(f"{x},{y}" for x, y in pts)
        self.parts.append(f'<path d="{p}" fill="none" stroke="{color}" stroke-width="1.4" marker-end="url(#ah)"{d}/>')
        if label and lx is not None:
            self.parts.append(f'<text x="{lx}" y="{ly}" text-anchor="middle" font-size="{fs}" fill="{color}">{html.escape(label)}</text>')
    def actor(self, x, y, name):
        # stick figure centered at x, head at y
        self.parts.append(f'<circle cx="{x}" cy="{y}" r="9" fill="#fff" stroke="#1f2937" stroke-width="1.4"/>')
        self.parts.append(f'<line x1="{x}" y1="{y+9}" x2="{x}" y2="{y+34}" stroke="#1f2937" stroke-width="1.4"/>')
        self.parts.append(f'<line x1="{x-14}" y1="{y+18}" x2="{x+14}" y2="{y+18}" stroke="#1f2937" stroke-width="1.4"/>')
        self.parts.append(f'<line x1="{x}" y1="{y+34}" x2="{x-12}" y2="{y+52}" stroke="#1f2937" stroke-width="1.4"/>')
        self.parts.append(f'<line x1="{x}" y1="{y+34}" x2="{x+12}" y2="{y+52}" stroke="#1f2937" stroke-width="1.4"/>')
        self.text(x, y+66, name, fs=11, anchor="middle", bold=True)
    def ellipse(self, cx, cy, rx, ry, lines, fill="#f8fafc", fs=11):
        self.parts.append(f'<ellipse cx="{cx}" cy="{cy}" rx="{rx}" ry="{ry}" fill="{fill}" stroke="#1f2937" stroke-width="1.3"/>')
        if isinstance(lines, str): lines = [lines]
        n = len(lines); lh = fs+3
        y0 = cy - (n-1)*lh/2 + fs/3
        for i, t in enumerate(lines):
            self.parts.append(f'<text x="{cx}" y="{y0+i*lh:.1f}" text-anchor="middle" font-size="{fs}">{html.escape(t)}</text>')
    def render(self):
        defs = ('<defs><marker id="ah" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto">'
                '<path d="M0,0 L10,4 L0,8 z" fill="#1f2937"/></marker></defs>')
        body = "\n".join(self.parts)
        return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {self.w} {self.h}" width="100%" '
                f'font-family="Helvetica, Arial, sans-serif" role="img" aria-label="{html.escape(self.title)}">{defs}\n{body}\n</svg>')

figs = {}

# ---------- Figure 1: system context, Themis at the core ----------
s = SVG(980, 640, "System context")
s.box(20, 20, 940, 600, "", fill="#fbfbfb", stroke="#9ca3af", dash=True)
s.text(34, 44, "Enterprise VM (same host: Themis estate and the AI harness)", fs=12, bold=True, fill="#6b7280")
# Themis core
s.box(330, 90, 320, 300, "", fill="#eef2ff", stroke="#3730a3", rx=14)
s.text(490, 114, "THEMIS — system of record for security truth", fs=14, anchor="middle", bold=True, fill="#312e81")
s.box(350, 128, 130, 44, ["Registry", "Products · Releases"], fill="#fff", fs=11)
s.box(500, 128, 130, 44, ["Evidence", "SBOM intake"], fill="#fff", fs=11)
s.box(350, 190, 130, 46, ["Knowledge", "CVE correlation"], fill="#fff", fs=11)
s.box(500, 190, 130, 46, ["Governance", "Findings · Proposals", "Positions"], fill="#fff", fs=10)
s.arrow(565, 172, 565, 190)                      # evidence -> governance lane (down)
s.arrow(415, 172, 415, 190)                      # registry ids -> knowledge
s.arrow(480, 213, 500, 213, "events", lx=490, ly=205, fs=9)
s.box(350, 288, 130, 36, ["Communication", "(not on demo path)"], fill="#f3f4f6", fs=10, stroke="#9ca3af")
s.box(500, 288, 130, 36, ["Intelligence", "(AI off for demo)"], fill="#f3f4f6", fs=10, stroke="#9ca3af")
s.box(350, 334, 280, 44, ["PostgreSQL, one database per context", "auth: API keys · scopes admin | read | product:<id>"], fill="#fff", fs=10)
# Harness on the left
s.box(40, 90, 250, 300, "", fill="#ecfdf5", stroke="#065f46", rx=14)
s.text(165, 114, "AI HARNESS (themis-ai-runtime)", fs=13, anchor="middle", bold=True, fill="#064e3b")
s.text(165, 130, "bounded execution infrastructure", fs=11, anchor="middle", italic=True, fill="#065f46")
s.box(55, 142, 220, 40, ["L1–L11 governed chain", "instructions → ratchet"], fill="#fff", fs=10)
s.box(55, 190, 220, 40, ["G1 deployment anchor rsys@6", "pins policy, skills, themis_contract"], fill="#fff", fs=10)
s.box(55, 245, 220, 40, ["L4 read seam → Themis API", "get_finding (governed-record)"], fill="#fff", fs=10)
s.box(55, 293, 220, 40, ["L5 execution · L6 record plane", "events + objects, hash-chained"], fill="#fff", fs=10)
s.box(55, 341, 220, 40, ["Model: local Ollama, tool-calling", "output = advisory data, never authority"], fill="#fff", fs=10, stroke="#9ca3af")
# read door: L4 seam -> Governance, routed in the lane between rows
s.path([(275, 265), (565, 265), (565, 238)], label="READ DOOR: HTTP, read-scope key, projected Finding only", lx=430, ly=258, fs=9.5)
# record plane + intake at bottom
s.box(55, 420, 220, 60, ["Harness record plane", "/srv/themis/rsys/state", "manifest · events.log · objects/"], fill="#fff", fs=10)
s.box(330, 420, 320, 60, ["themis-intake (Themis-owned CLI)", "tuple → intake.Resolve → Evidence View → Proposal", "reads the record locally, read-only"], fill="#fef3c7", fs=10, stroke="#92400e")
s.arrow(275, 450, 330, 450, "local", lx=302, ly=443)
s.arrow(165, 390, 165, 420, "L6 writes", lx=205, ly=409, fs=9.5)
s.path([(650, 440), (700, 440), (700, 226), (632, 226)])
s.text(708, 300, "Proposal over the", fs=9.5)
s.text(708, 312, "authenticated API", fs=9.5)
s.text(708, 324, "(product write key)", fs=9.5)
# humans on the right
s.actor(880, 110, "Decider")
s.text(880, 190, "acceptProposal", fs=10, anchor="middle")
s.text(880, 203, "(separate write key)", fs=10, anchor="middle")
s.arrow(862, 140, 632, 200, "accept → Position vN", lx=770, ly=160)
s.actor(880, 340, "Operator")
s.text(880, 420, "instantiates the task;", fs=10, anchor="middle")
s.text(880, 433, "runs themis-intake", fs=10, anchor="middle")
s.arrow(858, 372, 652, 462, "runs", lx=770, ly=410)
# legend
s.box(40, 510, 920, 90, "", fill="#fff", stroke="#9ca3af")
s.text(55, 532, "Boundary summary", fs=12, bold=True)
s.text(55, 550, "• The harness reaches Themis only through one HTTP read door; it never imports Themis and never initiates a Governance act.", fs=11)
s.text(55, 566, "• Themis never reads the harness record plane; only the Themis-owned themis-intake CLI does, locally, read-only.", fs=11)
s.text(55, 582, "• Security truth (Findings, Proposals, Positions) exists only inside Themis; the model's output is advisory data at every step.", fs=11)
figs["context"] = s.render()

# ---------- Figure 2: harness layers with the two doors ----------
s = SVG(980, 560, "Harness layers")
layers = [
    ("L1 Instructions", "governed instruction roots, directive policy"),
    ("L2 Context delivery", "four-class authority framing; Themis bytes never via L2"),
    ("L3 Context mgmt", "selection, budgets"),
    ("L4 Tool interface", "registry-v6, grants, uuid themis_scope; seam → governed-record"),
    ("L5 Execution", "sealed workspace; l5-transition / l5-op witnesses (W-M2)"),
    ("L6 Durable state", "hash-chained events, addressed objects, class→writer invariant (W-M1)"),
    ("L7 Orchestration", "workflow δ, phase gates, seal → egress → bind → COMPLETED"),
    ("L8 Subagents", "delegation templates, record-ref furniture"),
    ("L9 Skills", "remediate-dependency@4 = pinned composition, L9 mints no scope"),
    ("L10 Verification", "report-valid@2 contract; reconstructable PASS"),
    ("L11 Ratchet", "criteria, regression sets"),
]
y = 60
for name, desc in layers:
    fill = "#ecfdf5" if name.startswith(("L4", "L5", "L6", "L10")) else "#fff"
    s.box(180, y, 480, 36, "", fill=fill)
    s.text(190, y+22, name, fs=12, bold=True)
    s.text(338, y+22, desc, fs=9.5)
    y += 40
s.box(30, 60, 130, 436, ["G1", "Deployment", "anchor", "rsys@6", "", "pins every", "governed", "artifact incl.", "themis_contract"], fill="#fef3c7", fs=11)
s.box(720, 60, 230, 436, "", fill="#eef2ff", stroke="#3730a3")
s.text(835, 84, "THEMIS", fs=13, anchor="middle", bold=True, fill="#312e81")
s.text(835, 100, "(its own repo and estate)", fs=10, anchor="middle", italic=True)
s.box(735, 120, 200, 56, ["Governance API", "GET /findings/{id}", "POST proposals / accept"], fill="#fff", fs=10)
s.box(735, 190, 200, 44, ["Registry API", "GET /products/{id}"], fill="#fff", fs=10)
s.box(735, 250, 200, 70, ["Governance adapter: harness", "intake.Resolve (D-T-1..6)", "five-link replay (D-W-5)", "imports 4 read-only harness pkgs"], fill="#fff", fs=10)
s.box(735, 335, 200, 56, ["cmd/themis-intake", "human-operated bridge", "tuple in → Proposal out"], fill="#fef3c7", fs=10, stroke="#92400e")
s.box(735, 405, 200, 80, ["Walls", "exactly ONE Themis package", "imports the harness;", "harness imports no Themis;", "no writer, no exec"], fill="#fff", fs=10)
# doors
s.path([(660, 198), (698, 198), (698, 148), (735, 148)])
s.text(698, 174, "READ", fs=9, anchor="middle", bold=True); s.text(698, 185, "DOOR", fs=9, anchor="middle", bold=True)
s.path([(660, 278), (698, 278), (698, 285), (735, 285)])
s.text(698, 300, "DECISION", fs=9, anchor="middle", bold=True); s.text(698, 311, "DOOR", fs=9, anchor="middle", bold=True)
s.text(698, 323, "record read by", fs=8, anchor="middle"); s.text(698, 333, "themis-intake only", fs=8, anchor="middle")
s.text(490, 528, "G2 fact table: every fact names its establishing mechanism; the Themis row is the Position, established by acceptProposal.", fs=11, anchor="middle", italic=True)
figs["layers"] = s.render()

# ---------- Figure 3: use case diagram ----------
s = SVG(980, 520, "Use case diagram")
s.box(200, 20, 600, 480, "", fill="#fbfbfb", stroke="#9ca3af", dash=True)
s.text(500, 44, "Demo system boundary: Themis + AI harness on the enterprise VM", fs=12, anchor="middle", bold=True, fill="#6b7280")
s.actor(80, 90, "Operator")
s.actor(80, 250, "Decider")
s.actor(80, 400, "Model")
s.actor(910, 120, "Themis pipeline")
s.text(910, 200, "(Evidence → Knowledge", fs=9, anchor="middle")
s.text(910, 212, "→ Governance)", fs=9, anchor="middle")
s.actor(910, 330, "Reviewer")
uc = {}
def U(k, cx, cy, lines, fill="#f8fafc"):
    uc[k] = (cx, cy); s.ellipse(cx, cy, 120, 26, lines, fill=fill)
U("finding", 350, 90, ["UC1 Establish a real Finding", "SBOM → correlation → Finding UUID"])
U("commission", 350, 160, ["UC2 Commission governed work", "instantiate remediate-dependency@4"])
U("execute", 620, 160, ["UC3 Execute under anchor rsys@6", "read door · edit · report · L10 PASS"])
U("resolve", 350, 240, ["UC4 Resolve the execution", "themis-intake: tuple → evidence view"])
U("propose", 620, 240, ["UC5 Raise a Proposal", "human, evidence attached, trust inferred"])
U("accept", 570, 320, ["UC6 Accept the Proposal", "authenticated key → Position vN"])
U("h1", 350, 410, ["UC7 Substitute another Finding", "→ L4 denies (uuid scope / D-I-9)"], fill="#fee2e2")
U("h2", 650, 410, ["UC8 Verify A, egress B", "→ intake refuses, link-named"], fill="#fee2e2")
U("h3", 480, 470, ["UC9 Replay any Position cold", "record plane → same evidence"], fill="#dcfce7")
def L(x1, y1, x2, y2, dash=False, label=""):
    s.arrow(x1, y1, x2, y2, label, dash=dash, color="#4b5563")
L(100, 120, 230, 100)
L(100, 125, 230, 160)
L(100, 128, 230, 240)
L(100, 280, 450, 320)
L(100, 430, 230, 410)
L(890, 150, 470, 95)
L(890, 360, 600, 470)
L(890, 355, 770, 410)
s.arrow(470, 160, 500, 160, "", color="#4b5563")
s.text(485, 150, "«include»", fs=9, anchor="middle", fill="#4b5563")
s.arrow(470, 240, 500, 240, "", color="#4b5563")
s.text(485, 230, "«include»", fs=9, anchor="middle", fill="#4b5563")
s.arrow(620, 186, 620, 214, "", dash=True, color="#4b5563")
s.text(660, 203, "«precedes»", fs=9, anchor="middle", fill="#4b5563")
s.arrow(600, 266, 580, 294, "", dash=True, color="#4b5563")
s.text(548, 285, "«precedes»", fs=9, anchor="middle", fill="#4b5563")
s.arrow(720, 384, 720, 266, "", dash=True, color="#b91c1c")
s.text(768, 330, "«extends» (hostile)", fs=9, anchor="middle", fill="#b91c1c")
s.arrow(300, 384, 540, 186, "", dash=True, color="#b91c1c")
s.text(300, 300, "«extends» (hostile)", fs=9, anchor="middle", fill="#b91c1c")
figs["usecase"] = s.render()

# ---------- Figure 4: sequence diagram ----------
s = SVG(980, 900, "Sequence diagram")
lanes = ["Operator", "Themis pipeline", "Harness L7 (rsys@6)", "Model", "Themis Governance", "Record plane (L6)", "themis-intake", "Decider"]
xs = [70, 190, 330, 450, 580, 700, 820, 930]
for x, n in zip(xs, lanes):
    s.box(x-52, 20, 104, 40, n, fill="#f3f4f6", fs=10)
    s.parts.append(f'<line x1="{x}" y1="60" x2="{x}" y2="862" stroke="#9ca3af" stroke-width="1" stroke-dasharray="4,4"/>')
y = 90
def msg(a, b, label, dy=35, dash=False, color="#1f2937"):
    global y
    x1, x2 = xs[a], xs[b]
    s.arrow(x1, y, x2, y, "", dash=dash, color=color)
    s.text((x1+x2)/2, y-6, label, fs=9.5, anchor="middle", fill=color)
    y += dy
def note(a, label, color="#374151"):
    global y
    anchor = "end" if a >= 5 else "middle"
    s.text(965 if a >= 5 else xs[a], y-8, label, fs=9.5, anchor=anchor, fill=color, italic=True)
    y += 24
msg(0, 1, "1. register Product/Project/Release; upload CycloneDX SBOM")
msg(1, 4, "2. Knowledge correlates CVE → Governance opens Finding (UUID)")
msg(4, 0, "3. Finding UUID + provenance (runbook)", dash=True)
msg(0, 2, "4. instantiate remediate-dependency@4 {finding, dependency, advisory, target-version}")
msg(2, 5, "5. CREATED; materialized anchor bytes; governed hashes")
msg(2, 3, "6. compose phase ANALYZE (Finding UUID in context as data)")
msg(3, 2, "7. tool call get_finding(id=UUID)")
msg(2, 4, "8. L4 authorizes (uuid scope) → HTTP GET /findings/{id} (read key)")
msg(4, 2, "9. FindingView → projected (no positions/proposals) → governed-record", dash=True)
msg(2, 5, "10. l4-audit, model-turn, l2-delivery events")
msg(3, 2, "11. write_file go.mod / report.json; verify_report(report-valid@2)")
msg(2, 5, "12. l10-verification PASS (evaluation record, raw + canonical bytes)")
msg(2, 5, "13. l5-transition SEALED(task-complete) → EGRESSING; l5-op egress acknowledged A")
msg(2, 5, "14. artifact-bound A; lifecycle COMPLETED")
msg(0, 6, "15. themis-intake (anchor, task, seq) --stance --rationale")
msg(6, 5, "16. local read-only: manifest, events, objects")
note(6, "17. Resolve: D-T-1..6, five-link replay, reproducible PASS over the egressed bytes")
msg(6, 0, "18. Evidence View: model turns | artifact | verification (three facts)", dash=True)
msg(6, 4, "19. POST proposals {stance, rationale, evidence: harness-execution/v1, trust: inferred} (write key)")
msg(4, 6, "20. proposal_id", dash=True)
msg(7, 4, "21. POST proposals/{id}/accept (separate write key)")
note(4, "22. decider = key:<KeyID>; Position vN cites AcceptedProposalID")
msg(4, 7, "23. Position established", dash=True)
s.text(490, 884, "Every numbered step leaves a durable record either in Themis (Postgres) or in the harness record plane; step 17 is recomputed, never read as a claim.", fs=10, anchor="middle", italic=True)
figs["sequence"] = s.render()

# ---------- Figure 5: authority chain and three facts ----------
s = SVG(980, 300, "Authority chain")
chain = [("Model", ["produces the", "work / artifact"], "#f3f4f6"), ("Harness", ["verifies, records;", "reconstructable"], "#ecfdf5"),
         ("themis-intake", ["human invokes;", "derived evidence"], "#fef3c7"), ("Themis Proposal", ["proposer =", "authenticated human"], "#eef2ff"),
         ("Business Verif.", ["refs vouched", "by the Finding"], "#eef2ff"), ("Human acceptance", ["separate", "authenticated key"], "#eef2ff"), ("Position", ["Themis", "security truth"], "#c7d2fe")]
x = 20
for i, (n, d, f) in enumerate(chain):
    s.box(x, 40, 124, 70, [n] + d, fill=f, fs=9.5)
    if i < len(chain)-1: s.arrow(x+124, 75, x+138, 75)
    x += 138
s.text(490, 30, "Authority chain (D-I-5): the model is never the decision-maker; the harness is never a Governance actor.", fs=12, anchor="middle", bold=True)
s.box(20, 140, 300, 130, ["HARNESS FACT", "this task bound this artifact", "artifact-bound (l6) after", "l5 seal + egress witnesses"], fill="#ecfdf5", fs=11)
s.box(340, 140, 300, 130, ["VERIFICATION FACT", "L10 evaluated the exact bytes", "l10-verification + evaluation record;", "re-run by Themis, never trusted"], fill="#ecfdf5", fs=11)
s.box(660, 140, 300, 130, ["GOVERNANCE FACT", "an authorized human accepted it", "Position vN → AcceptedProposalID →", "harness-execution/v1 evidence"], fill="#eef2ff", fs=11)
s.text(490, 292, "Three facts, never collapsed into one \"approved report\" object.", fs=11, anchor="middle", italic=True)
figs["authority"] = s.render()

# ---------- Figure 6: deployment topology ----------
s = SVG(980, 460, "Deployment topology")
s.box(20, 20, 940, 420, "", fill="#fbfbfb", stroke="#9ca3af", dash=True)
s.text(34, 44, "Enterprise Linux VM", fs=12, bold=True, fill="#6b7280")
s.box(40, 70, 420, 330, "", fill="#eef2ff", stroke="#3730a3")
s.text(250, 94, "Themis estate (~/code/themis checkout, systemd units themis@<ctx>)", fs=11, anchor="middle", bold=True)
for i, (n, p) in enumerate([("registry", ":8082"), ("evidence", ":8081"), ("knowledge", ":8085"), ("governance", ":8083"), ("communication", ":8084"), ("intelligence", ":8086")]):
    col, row = i % 2, i // 2
    on = n in ("registry", "evidence", "knowledge", "governance")
    s.box(55 + col*200, 110 + row*52, 185, 40, [f"themis@{n}", p + ("  · on demo path" if on else "  · not on path")], fill="#fff" if on else "#f3f4f6", fs=10, stroke="#1f2937" if on else "#9ca3af")
s.box(55, 272, 385, 40, ["PostgreSQL: one database per context (+ bus, auth)"], fill="#fff", fs=10)
s.box(55, 322, 385, 64, ["THEMIS_AUTH_REQUIRED=1 on governance + registry; AI off", "keys (authadmin): read (harness) · product write (operator) · write (decider)", "deployed commit == commit named in themis_contract"], fill="#fff", fs=10)
s.box(500, 70, 440, 330, "", fill="#ecfdf5", stroke="#065f46")
s.text(720, 94, "Harness (~/code/themis-ai-runtime checkout, themis-run)", fs=11, anchor="middle", bold=True)
s.box(515, 110, 410, 56, ["Governed root: the checkout at the rsys@6 commit", "policies/ · instructions/ · policies/themis/contract.json"], fill="#fff", fs=10)
s.box(515, 176, 410, 66, ["Deployment root /srv/themis/rsys", "anchor rsys@6 (ACTIVE) · anchors.json (append-only; rsys@5 withdrawn)", "state/ (record plane) · artifacts/ · provider/ · mirror"], fill="#fff", fs=10)
s.box(515, 252, 410, 56, ["rsys@6 pins: constitution.state (W-M1) · tool_registry v6 · skill_catalog (@4)", "themis_contract = sha256(contract.json): URLs, spec hashes, Themis commit"], fill="#fef3c7", fs=10, stroke="#92400e")
s.box(515, 318, 410, 68, ["Demo module mirror: real Go module with a real CVE", "CycloneDX SBOM of the same module fed to Evidence", "themis-intake binary (built from the Themis repo, pinned harness dep)"], fill="#fff", fs=10)
s.arrow(500, 206, 462, 206, "loopback", lx=481, ly=196, fs=9)
s.text(490, 424, "Same host: the record plane is local to themis-intake; only the Proposal and the reads cross a socket, on loopback.", fs=11, anchor="middle", italic=True)
figs["deploy"] = s.render()

# ---------- HTML ----------
css = """
@page { size: A4; margin: 16mm 14mm; }
body { font-family: Helvetica, Arial, sans-serif; color: #111827; font-size: 11.5pt; line-height: 1.42; margin: 0; }
h1 { font-size: 24pt; margin: 0 0 4pt; }
h2 { font-size: 16pt; margin: 22pt 0 8pt; border-bottom: 1.5px solid #1f2937; padding-bottom: 3pt; }
h3 { font-size: 12.5pt; margin: 14pt 0 6pt; }
p { margin: 5pt 0; }
.meta { color: #4b5563; font-size: 10pt; }
.fig { margin: 8pt 0 4pt; page-break-inside: avoid; }
.cap { font-size: 9.5pt; color: #4b5563; margin: 2pt 0 10pt; }
table { border-collapse: collapse; width: 100%; font-size: 10pt; margin: 6pt 0 10pt; page-break-inside: auto; }
th, td { border: 1px solid #9ca3af; padding: 4pt 6pt; vertical-align: top; text-align: left; }
th { background: #f3f4f6; }
tr { page-break-inside: avoid; }
pre { background: #f8fafc; border: 1px solid #d1d5db; padding: 6pt 8pt; font-size: 8.8pt; line-height: 1.3; white-space: pre-wrap; page-break-inside: avoid; }
code { font-family: Menlo, Consolas, monospace; font-size: 9.5pt; }
.pb { page-break-before: always; }
ul { margin: 4pt 0 6pt 18pt; padding: 0; }
li { margin: 2pt 0; }
.callout { border-left: 4px solid #3730a3; background: #eef2ff; padding: 6pt 10pt; margin: 8pt 0; }
"""

def fig(key, n, caption):
    return f'<div class="fig">{figs[key]}</div><div class="cap">Figure {n}. {html.escape(caption)}</div>'

doc = f"""<!DOCTYPE html><html><head><meta charset="utf-8"><title>Themis AI Runtime — Demo Architecture and Use Cases</title><style>{css}</style></head><body>
<h1>Themis AI Runtime — Demo Architecture and Use Cases</h1>
<p class="meta">Themis at the core, the AI harness as bounded execution infrastructure. Prepared 2026-09-26 from the locked decision records
(<code>openspec/changes/themis-v0</code> D-T-1..8, <code>l5-witness-events</code> D-W-1..6, <code>themis-integration</code> D-I-1..9). Status: architecture locked; implementation milestones I-M0..I-M5 pending.</p>

<div class="callout"><strong>The claim the demo defends.</strong> Themis remains the authority for security truth while the harness remains bounded execution infrastructure.
A real Finding in Themis is remediated by governed AI execution; the execution is verified and independently reconstructed; a human proposes and a different human decides; the model never touches the decision and the harness never writes security truth.</div>

<h2>1. System context</h2>
<p>Themis is the system of record: six bounded-context services over PostgreSQL, deployed on the enterprise VM. The harness is the eleven-layer governed execution chain with one deployment anchor.
They meet at exactly two doors: a <em>read door</em> (the harness reads a projected Finding over HTTP) and a <em>decision door</em> (a Themis-owned CLI reads the harness record locally and raises a Proposal to Governance).</p>
{fig("context", 1, "System context on the enterprise VM: Themis at the core, the harness beside it, humans on the outside, two doors between them.")}

<h2 class="pb">2. Harness architecture and where Themis plugs in</h2>
<p>Every layer already exists and is operationally proven under anchors rsys@3..@5. The integration adds no layer: it fills the seams the harness left for Themis and adds L5-owned witnesses to the record so that production can be replayed by the layer that executed it.</p>
{fig("layers", 2, "Harness layers L1–L11 with the G1 anchor; the read door leaves through L4, the decision door enters the record plane from Themis's side only.")}
<h3>2.1 Locked boundaries that shape the diagram</h3>
<ul>
<li><strong>Read door (D-I-3).</strong> <code>ThemisSeam.Read</code> is an HTTP client. <code>get_finding</code> serves id, release, faultline, CVE, stage, components; positions and proposals are stripped. The anchor pins <code>themis_contract</code> (endpoints, two OpenAPI spec hashes, Themis commit), never live data. The read-scope key lives only in the harness environment.</li>
<li><strong>Decision door (D-I-1, D-I-5, D-I-6).</strong> <code>cmd/themis-intake</code> is Themis-owned, runs on the same host, takes the execution tuple only, resolves it through D-T-1..6, and raises a Governance Proposal as an authenticated human. <code>acceptProposal</code> by a separate authenticated key creates the Position.</li>
<li><strong>Witnesses (D-W-2, D-W-3, D-W-6).</strong> L5 writes one <code>l5-transition</code> per state-machine edge and one <code>l5-op</code> per subprocess and per egress through a writer-fixed handle; the sink enforces a closed class→writer table, folded into the constitution hash (D-W-4).</li>
<li><strong>Walls (D-I-7).</strong> Exactly one Themis package imports four read-only harness packages; the harness imports no Themis code; neither side can execute or write through the other.</li>
</ul>

<h2 class="pb">3. Demo use cases</h2>
<p>Six primary use cases form the happy path; three hostile use cases are the refusals a reviewer can ask for live. Actors: the Operator (commissions work, runs intake), the Decider (accepts), the Model (advisory only), the Themis pipeline, and the Reviewer.</p>
{fig("usecase", 3, "Use case diagram for the thirty-minute live demo. Red: hostile demonstrations that must refuse. Green: cold replay of a Position from the record.")}
<table>
<tr><th>UC</th><th>Actor</th><th>Goal</th><th>Enforced by</th><th>Locked in</th></tr>
<tr><td>UC1</td><td>Operator, Themis pipeline</td><td>A real Finding exists for a real vulnerable Go module (no manual row)</td><td>Evidence → Knowledge → Governance</td><td>D-I-4, D-I-8</td></tr>
<tr><td>UC2</td><td>Operator</td><td>Commission <code>remediate-dependency@4</code> with finding UUID, dependency, CVE, target version</td><td>L9 instantiation, catalog pin</td><td>D-I-4</td></tr>
<tr><td>UC3</td><td>Harness, Model</td><td>Execute under rsys@6: read Finding as governed record, edit, report, L10 PASS, seal, egress, bind, COMPLETED</td><td>L1–L11, G1, L5 witnesses</td><td>D-I-3, D-W-2/3</td></tr>
<tr><td>UC4</td><td>Operator</td><td>Resolve the tuple: anchor registered, five-link production chain, reproducible PASS over egressed bytes; evidence view</td><td><code>themis-intake</code>, <code>intake.Resolve</code></td><td>D-T-1..6, D-W-5</td></tr>
<tr><td>UC5</td><td>Operator</td><td>Raise a Proposal with immutable <code>harness-execution/v1</code> evidence, trust derived as inferred</td><td>Governance API, Business Verification</td><td>D-I-5</td></tr>
<tr><td>UC6</td><td>Decider</td><td>Accept → Position vN with decider <code>key:&lt;KeyID&gt;</code></td><td><code>acceptProposal</code>, EDR-SECURITY-01 D10</td><td>D-I-6</td></tr>
<tr><td>UC7</td><td>Model (hostile)</td><td>Ask for another Finding</td><td>L4 uuid scope; exact subject with D-I-9</td><td>D-I-4, D-I-9</td></tr>
<tr><td>UC8</td><td>Model (hostile)</td><td>Verify report A, rewrite to B, egress B</td><td>intake refuses "verified bytes are not the bound artifact"</td><td>D-T-5 (proven T-M3)</td></tr>
<tr><td>UC9</td><td>Reviewer</td><td>Replay any Position cold from the record plane and get the same evidence</td><td>Position → AcceptedProposalID → tuple → record</td><td>D-I-6, D-T-4</td></tr>
</table>

<h2 class="pb">4. The demo flow, step by step</h2>
{fig("sequence", 4, "Sequence of the live demo. Solid arrows are acts; dashed arrows are returns. Steps 1–3 are done before the session and shown as evidence; steps 4–23 run live.")}

<h2 class="pb">5. Authority chain and the three facts</h2>
{fig("authority", 5, "Who may do what (D-I-5), and the three facts the evidence view keeps separate (proposal §Why).")}

<h2>6. Expected outputs, per step</h2>
<p>What the audience sees on screen, and what durable record it corresponds to. Identifiers below are illustrative shapes, not real values.</p>
<table>
<tr><th>Step</th><th>Command / act</th><th>Visible output</th><th>Durable record</th></tr>
<tr><td>UC1</td><td><code>vm-verify.sh &lt;release&gt;</code>; <code>GET /findings?release=…&amp;faultline=…</code></td><td>Finding <code>id</code> (UUID), <code>cve</code>, <code>components[]</code> with the vulnerable PURL, <code>stage: identified</code>, no positions</td><td>Governance DB; runbook provenance (SBOM hash, release id, feed source)</td></tr>
<tr><td>UC2</td><td><code>themis-instantiate remediate-dependency@4 …</code></td><td>Envelope path; composition hash; grant with <code>themis_scope: [uuid]</code></td><td>Envelope file; L6 CREATED event with governed hashes</td></tr>
<tr><td>UC3</td><td><code>themis-run -deploy /srv/themis/rsys …</code></td><td><code>status=COMPLETED verdict=VERIFIED artifact=sha256:…</code>; audit lines for <code>get_finding authorized</code>, <code>verify_report authorized</code></td><td>events.log: l2-delivery, l4-audit, model-turn, l10-verification PASS, l5-transition ×N, l5-op ×N incl. egress acknowledged, artifact-bound, lifecycle COMPLETED</td></tr>
<tr><td>UC4</td><td><code>themis-intake --anchor … --task … --seq …</code></td><td>Evidence view (below); <code>production_witness: l5-witnessed</code>; <code>reconstructed_outcome: PASS</code></td><td>None written by Themis at this step; the CLI's rendering is presentation</td></tr>
<tr><td>UC5</td><td>same command with <code>--stance --rationale</code></td><td><code>proposal_id</code>; evidence echoed back by <code>GET /findings/{{id}}</code> with <code>proposer_kind: human</code>, <code>evidence_trust: inferred</code></td><td><code>finding_proposals</code> row with immutable <code>evidence</code> JSONB</td></tr>
<tr><td>UC6</td><td><code>POST …/accept</code> with the decider's key</td><td><code>current_position.version: 1</code>, <code>actor_kind: human</code>, <code>actor_id: key:…</code>, inputs cite the proposal</td><td>Position v1 row; outbox event for Communication</td></tr>
<tr><td>UC7</td><td>model calls <code>get_finding</code> with another UUID</td><td>Tool result: deterministic denial (class-only); walk continues or fails per workflow</td><td>l4-audit <code>Decision: denied</code></td></tr>
<tr><td>UC8</td><td>walk that verifies A then writes B; then <code>themis-intake</code></td><td><code>verification-refused: verified bytes are not the bound artifact (no member of egress … has hash …)</code></td><td>Harness record COMPLETED; no Proposal exists</td></tr>
<tr><td>UC9</td><td><code>themis-intake --inspect &lt;position&gt;</code> (or the tuple again)</td><td>Same evidence view, byte-for-byte identities</td><td>Read-only</td></tr>
</table>

<h3>6.1 Evidence view, as rendered by themis-intake (shape)</h3>
<pre>{{
  "execution": {{ "anchor_hash": "0d…", "anchor_name": "rsys", "anchor_version": 6, "anchor_state_at_intake": "active",
                  "task_id": "demo-remediate-0001", "constitution_hash": "…", "harness_module": "…/src/harness v0.0.0-…" }},
  "model_turns": [ {{ "seq": 7, "object_id": "sha256:…" }}, {{ "seq": 12, "object_id": "sha256:…" }} ],
  "artifact":  {{ "object_id": "sha256:…", "artifact_bound_seq": 41, "completed_seq": 44,
                  "verified_member_path": "report.json", "verified_member_sha256": "…",
                  "production_witness": "l5-witnessed",
                  "l5": {{ "sealed_seq": 36, "seal_reason": "task-complete", "egressing_seq": 37, "egress_op_seq": 39, "egress_outcome": "acknowledged" }} }},
  "verification": {{ "seq": 31, "contract": "report-valid@2", "contract_sha256": "bbc1…", "contract_state_at_intake": "active",
                     "verifier_audit_seq": 30, "reconstructed_outcome": "PASS", "reconstruction_consistent": true, "checks": 9 }},
  "trust_class": "inferred",
  "business_verification_refs": [ "pkg:golang/…@v1.x", "CVE-20xx-xxxx" ]
}}</pre>

<h3>6.2 Refusals the audience may ask to see</h3>
<table>
<tr><th>Ask</th><th>Refusal text (class)</th><th>Where</th></tr>
<tr><td>Tuple names a different anchor</td><td><code>execution not referencable: the record identifies deployment anchor …</code></td><td>intake D-T-1</td></tr>
<tr><td>Anchor never registered on Themis's checkout</td><td><code>deployment-refused: … no longer registered</code></td><td>intake D-T-2</td></tr>
<tr><td>Anchor withdrawn after the run</td><td>proceeds; <code>anchor_state_at_intake: withdrawn</code></td><td>intake D-T-6</td></tr>
<tr><td>Missing or forged L5 witness</td><td><code>artifact-provenance-refused: missing l5-op egress witness …</code> (first failed link)</td><td>intake D-W-5</td></tr>
<tr><td>Verify A, egress B</td><td><code>verification-refused: verified bytes are not the bound artifact</code></td><td>intake D-T-5</td></tr>
<tr><td>Event claims PASS over invalid bytes</td><td><code>verification-refused: no reproducible PASS (reconstruction inconsistent: canonicalization)</code></td><td>intake D-T-5</td></tr>
<tr><td>Model requests another Finding</td><td>L4 denial, audited</td><td>L4 D-I-4 / D-I-9</td></tr>
<tr><td>Accept without a principal</td><td>recorded as <code>dev:</code>; the demo asserts none exist</td><td>Governance D-I-6</td></tr>
</table>

<h2 class="pb">7. Deployment topology for the demo</h2>
{fig("deploy", 6, "One VM, two checkouts, one anchor (rsys@6), three keys. The contract pins the interface; the data is live.")}
<h3>7.1 Host-act order (D-I-8)</h3>
<ol>
<li><strong>Themis side:</strong> auth required on Governance and Registry; three keys issued with <code>authadmin</code>; deployed commit checked against <code>contract.json</code>; demo Finding established through the pipeline (fixture feed if OSV is unreachable).</li>
<li><strong>Harness side:</strong> W-M1..W-M3 and the module rename landed and proven hermetically; rsys@6 minted on the host from the host's tree; opened and validated; rsys@5 withdrawn; demo execution.</li>
<li><strong>Decision side:</strong> operator raises through <code>themis-intake</code>; decider accepts; no <code>dev:</code> witness.</li>
</ol>

<h2>8. Deliberately out of scope</h2>
<ul>
<li>Vulnerability-feed ingestion beyond the one demo advisory; Knowledge Builder; CVE enrichment.</li>
<li>Themis's Intelligence Gateway (AI proposals) — off for the demo so the only proposal is the human's.</li>
<li>Automated acceptance; any Position created by the harness or the model.</li>
<li>Multi-skill pipelines; harness-to-Themis traffic other than the single read door.</li>
<li>Process authentication and key custody beyond Themis's authentication administration.</li>
</ul>

<h2>9. Milestones to reach the demo</h2>
<table>
<tr><th>Milestone</th><th>Repo</th><th>Lands</th></tr>
<tr><td>I-M0</td><td>harness</td><td>Module rename to <code>github.com/tofchaliss/themis-ai-runtime/src/harness</code>; constitution hashes asserted unchanged</td></tr>
<tr><td>W-M1</td><td>harness</td><td>Sink-enforced class→writer invariant folded into the L6 constitution hash</td></tr>
<tr><td>W-M2</td><td>harness</td><td>L5 emission handle; full-machine <code>l5-transition</code>; two-form <code>l5-op</code>; AST wall</td></tr>
<tr><td>I-M1</td><td>harness</td><td>HTTP read seam with projection; <code>themis_contract</code>; registry v6 uuid scope; <code>remediate-dependency@4</code></td></tr>
<tr><td>I-M2</td><td>harness</td><td>Real five-link record fixture with provenance metadata</td></tr>
<tr><td>I-M3</td><td>Themis</td><td>EDR + phase3 change; evidence on proposals; <code>adapters/harness</code> with intake and five-link replay; <code>cmd/themis-intake</code>; walls; fixture tests</td></tr>
<tr><td>I-M4</td><td>harness</td><td><code>src/themis</code> dissolved; Wall 1 rewritten</td></tr>
<tr><td>I-M5</td><td>host</td><td>Keys, Finding via pipeline, single rsys@6 mint, demo execution, two human acts, Addendum G, reviews, archive</td></tr>
</table>
<p class="meta">Source: <code>docs/demo/themis-demo-architecture.html</code> (generated); decisions in <code>openspec/changes/{{themis-v0,l5-witness-events,themis-integration}}/design.md</code>.</p>
</body></html>"""

os.makedirs(os.path.dirname(OUT), exist_ok=True)
with open(OUT, "w") as f:
    f.write(doc)
print("wrote", OUT, len(doc), "bytes")
