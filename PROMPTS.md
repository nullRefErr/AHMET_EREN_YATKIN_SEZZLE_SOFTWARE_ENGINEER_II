# AI Prompts Log

Required by the assignment: *"Share any prompts that you used in your work."*

Tool: Claude Code (CLI), model Claude Opus 5.

Every prompt I typed is here, verbatim and in order. The one thing not reproduced is the
assignment e-mail itself, quoted at entry 2 — it is Sezzle's document, not my prompt.

Two subagents were written during the work and given standing instructions of their own: `Worker`
builds a phase test-first, `Officer` audits the result against the brief and `CLAUDE.md`. Those
instructions are prompts too, and they are in `.claude/agents/`.
Turkish prompts are followed by an English translation.

---

## 1

> Bu proje bir işe giriş projesi olacak, sana detayları md dökümanı aracılığıyla veriyorum, bu geliştirmeler için önce bir spec dosyası çıkar. /sc:brainstorm kullan /ponytail:ponytail kullan

*"This project will be a job-application project. I'm giving you the details through the md
document. For these developments, first produce a spec file. Use /sc:brainstorm. Use
/ponytail:ponytail."*

Attached: `CALCULATOR_MONOREPO_PLAN.md` — a technical guide prepared beforehand.

---

## 2

The Sezzle assignment e-mail, pasted verbatim as context. Not reproduced here — it is the
client's document.

---

## 3

> ödev isterleri öncelikli olacak şekilde spec dosyasını güncelleyelim, tek servis istiyorsa bunu dikkate alalım, prompts.md oluştur, verilen her promptu oraya inject et

*"Let's update the spec file so the assignment requirements take priority. If it wants a single
service, let's take that into account. Create prompts.md, and inject every prompt given into it."*

---

## 4

> First lets create a claude.md file for rules and boundries. your first couple of rules are;
>
> * use test driven development method,
> * Always apply software principles such as YAGNI, KISS, SOLID, DRY to write clean and readable codes
> * Obey the structure rules and design pattern boundries
>    * Apply repo pattern for project structure

---

## 5

> update the prompts.md file by cleaning outcome and other unnecessary explanations keep only the prompts

---

## 6

> create phases according to spec file, keep the rules from claude.md in mind /ponytail

---

## 7

> 1) yes
> 2) use Gin
> 3) add in-memory cache
> 4) use repo pattern without database access layer, simple design pattern code should be easy to read and clean

---

## 8

> keep going

---

## 9

> Update claude.md file with given rules and keep in mind our rules, if rules againsg eachother use most reasonable one

Attached: a Go engineering rule set with stable IDs and a "Writing Functions Best Practices"
checklist. It is reproduced in `CLAUDE.md`, not here.

---

## 10

> then keep goin to create workflow file

---

## 11

> write all the phases from F3 to end

---

## 12

> create an agent and name it "Officer" to investigate written workflow file according to given job descriptions and claude.md file, use /ponytail /code-review to keep in track and remove unnecessary sayings, code examples, comments and so on.

---

## 13

> did you update the prompts.md ?

---

## 14

> create an agent named "Worker" to start implementing phases according to the workflow file. use help of the /sc:implement start from F0

---

## 15

> do git init and name this project AHMET_EREN_YATKIN_SEZZLE_SOFTWARE_ENGINEER_II

---

## 16

> after this git commands start the agent/sc:agent create an agent named "Worker" to start implementing phases according to the workflow file. use help of the /sc:implement start from F0

---

## 17

> we ran a new agent called worker to develop phases, I want you to run officer against it to keep it in track, give directions according to the claude.md files and strucres and job description

---

## 18

> tabi çevirisini yapalım

*"Yes, let's translate it." — SPEC.md and the build record, Turkish to English.*

---

## 19

> first of all I want you to write conventional commits, split written code into a logical parts and commit these, you could follow feature/branch approach also

---

## 20

> check if there is any leftover phases

---

## 21

> you can create a repo and push this

---

## 22

> run the project and lets do a manuel test

---

## 23

> işlemlerde yanlışlık var, 75 + 52 - 30 = 22 çıkıyor

*"The operations are wrong, 75 + 52 - 30 gives 22."*

---

## 24

> sqrt ve percentage de test et

*"Test sqrt and percentage as well."*

---

## 25

> Check frontend to be able to render mobile view

---

## 26

> check given job description and codebase aaccording to requirements, I caught one for example, add coverage, have you added ?

---

## 27

> wait for officer's report then fix all of them

---

## 28

> gerek yok, hepsini commitleyip push yapalım

*"No need, let's commit and push everything."*

---

<!-- Append each new prompt below, verbatim. Prompts only — no commentary. -->
