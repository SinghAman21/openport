export type Signal = "alert" | "active" | "muted" | "protected"
export type Frame = {
  label: string
  title: string
  command: string
  rows: { name: string; detail: string; signal: Signal }[]
  output: string
}

export const guides = [
  {
    slug: "conflict", number: "01", category: "DEBUGGING", time: "2 MIN READ", title: "Port 3000 is taken. Now what?",
    excerpt: "The error tells you where it hurts, not who's responsible. Find the owner before you stop anything.",
    lead: "Your app refuses to start. You could kill every Node process and hope for the best. Or you could ask one better question: what, exactly, is listening on :3000?",
    promise: "Find the right process. Keep everything else running.",
    command: "openport check 3000", commandNote: "Inspect the port before you touch the process.",
    outcome: "One conflict resolved. Your database, cache and other projects stay exactly where they were.",
    steps: [
      { title: "Locate the owner", body: "Run openport check 3000 or open the Ports view. You'll see the owning process, PID and bind address. A port number alone is not enough reason to stop a process." },
      { title: "Make the call", body: "If the listener is a stale dev server, select it in the Ports view and press x. OpenPort shows a confirmation before it stops anything. If it belongs to another project, leave it alone." },
      { title: "Start clean", body: "With the stale process gone, port 3000 is available to your app. Protect important listeners like your database so a future cleanup cannot catch them by mistake." },
    ],
    frames: [
      { label: "THE BLOCK", title: "Your app can't bind :3000", command: "npm run dev", rows: [{ name: ":3000", detail: "EADDRINUSE", signal: "alert" }, { name: ":5432", detail: "database · running", signal: "muted" }, { name: ":6379", detail: "cache · running", signal: "muted" }], output: "The error gives you a port. Not a process." },
      { label: "THE ANSWER", title: "OpenPort names the owner", command: "openport check 3000", rows: [{ name: ":3000", detail: "node · PID 5678", signal: "alert" }, { name: ":5432", detail: "postgres · protected", signal: "protected" }, { name: ":6379", detail: "redis · running", signal: "muted" }], output: "You decide whether this listener is safe to stop." },
      { label: "THE FIX", title: "Confirm. Stop. Continue.", command: "openport  →  select :3000  →  x", rows: [{ name: ":3000", detail: "available for your app", signal: "active" }, { name: ":5432", detail: "postgres · protected", signal: "protected" }, { name: ":6379", detail: "redis · running", signal: "active" }], output: "Illustration only: stopping a process requires confirmation in the CLI." },
    ],
  },
  {
    slug: "stack", number: "02", category: "ORGANIZATION", time: "3 MIN READ", title: "Make your local stack recognizable.",
    excerpt: "Replace a wall of port numbers with names that mean something to your project.",
    lead: "3000, 5432, 6379. A familiar set of numbers, until you open a second repo and forget which process belongs to which environment. Give your machine a vocabulary.",
    promise: "Know what's yours at a glance, even after a restart.",
    command: "openport --view aliases", commandNote: "Start naming the services you already run.",
    outcome: "Your stack reads like a map instead of a list of numbers. Project config makes that map portable with the repo.",
    steps: [
      { title: "Name what you know", body: "Open the Aliases view and name port 3000 frontend, 5432 database and 6379 cache. The names appear beside live listeners in the Ports view." },
      { title: "Put the map in the repo", body: "Create .openport.yml with aliases and expected_ports. OpenPort discovers the file from your working directory or a parent, even when you launch from a subdirectory." },
      { title: "Set a guardrail", body: "Add 5432 to the project's protected list. The ordinary kill path will refuse it; overriding protection requires extra confirmation in the interface." },
    ],
    frames: [
      { label: "BEFORE", title: "Just numbers, no context", command: "openport", rows: [{ name: ":3000", detail: "node", signal: "muted" }, { name: ":5432", detail: "postgres", signal: "muted" }, { name: ":6379", detail: "redis-server", signal: "muted" }], output: "Which process belongs to this project?" },
      { label: "NAME IT", title: "Give the stack a language", command: "openport --view aliases", rows: [{ name: ":3000", detail: "frontend · node", signal: "active" }, { name: ":5432", detail: "database · postgres", signal: "active" }, { name: ":6379", detail: "cache · redis", signal: "active" }], output: "Aliases make each listener instantly recognizable." },
      { label: "KEEP IT", title: "A map that travels with the repo", command: ".openport.yml  →  protected: [5432]", rows: [{ name: "frontend", detail: ":3000 · expected", signal: "active" }, { name: "database", detail: ":5432 · protected", signal: "protected" }, { name: "cache", detail: ":6379 · expected", signal: "active" }], output: "Names, expected ports and protection live in project config." },
    ],
  },
  {
    slug: "free-port", number: "03", category: "WORKFLOW", time: "2 MIN READ", title: "Find a free port without guessing.",
    excerpt: "Stop trying 3001, then 3002, then 3003. Ask for a port you can actually bind.",
    lead: "A second service needs somewhere to run. Picking the next number feels fast—until it's occupied, reserved, or unavailable for binding. A real free port should pass more than a glance at the listener list.",
    promise: "Get a usable port, not another surprise.",
    command: "openport free --near 4000", commandNote: "Ask for an available port near your usual range.",
    outcome: "The output is a bare port number. Use it in your local app or capture it in a script; reserve it if that service should keep it.",
    steps: [
      { title: "Ask near your range", body: "Run openport free --near 4000. OpenPort skips ports that are listening, reserved, or protected, then tests whether the candidate can actually be bound." },
      { title: "Use the result", body: "The command prints a bare number on stdout, which is easy to pass into your own app configuration or capture in a script. The port shown here is an illustration, not a promise about your machine." },
      { title: "Keep the slot", body: "If that port belongs to a specific service, reserve it in OpenPort. Future free-port suggestions will not hand it to something else." },
    ],
    frames: [
      { label: "THE GUESS", title: "The next number isn't always free", command: "try :4000, then :4001, then…", rows: [{ name: ":4000", detail: "already listening", signal: "alert" }, { name: ":4001", detail: "reserved", signal: "protected" }, { name: ":4002", detail: "not checked yet", signal: "muted" }], output: "A gap in the list isn't a guarantee." },
      { label: "THE CHECK", title: "Ask the machine, not your memory", command: "openport free --near 4000", rows: [{ name: ":4000", detail: "skip · listening", signal: "alert" }, { name: ":4001", detail: "skip · reserved", signal: "protected" }, { name: ":4002", detail: "bind check passed", signal: "active" }], output: "Candidates must actually be bindable." },
      { label: "THE SLOT", title: "A place for the new service", command: "openport free --near 4000  →  4002", rows: [{ name: ":4000", detail: "existing service", signal: "muted" }, { name: ":4001", detail: "reserved", signal: "protected" }, { name: ":4002", detail: "available · use this one", signal: "active" }], output: "Example result only. Your machine may return another port." },
    ],
  },
  {
    slug: "blueprint", number: "04", category: "RELIABILITY", time: "3 MIN READ", title: "Know when your environment drifts.",
    excerpt: "A stack can look alive and still be wrong. Compare what's running with what the project expects.",
    lead: "Your frontend is up, the database is running, and yet something still feels off. The problem may be a missing service—or the right process on the wrong port. Make the expected state explicit.",
    promise: "See the mismatch before you lose an afternoon to it.",
    command: "openport apply web", commandNote: "After creating a blueprint named web in the UI.",
    outcome: "Applying is read-only: it tells you what's missing or displaced and suggests a next step. It never starts or kills a process for you.",
    steps: [
      { title: "Describe the expected environment", body: "In the Blueprints view, create a blueprint called web with the services, ports and health probes your project expects." },
      { title: "Compare against what is running", body: "Apply the blueprint in the UI or run openport apply web. OpenPort reports which services are up, absent, or blocked by a different listener." },
      { title: "Fix with intention", body: "The report is a plan, not an automation run. Choose what to start, move or leave alone yourself. OpenPort won't kill or launch anything just because a blueprint says so." },
    ],
    frames: [
      { label: "EXPECTED", title: "What this project needs", command: "blueprint: web", rows: [{ name: "frontend", detail: ":3000 · expected", signal: "active" }, { name: "database", detail: ":5432 · expected", signal: "active" }, { name: "worker", detail: ":9000 · expected", signal: "muted" }], output: "A blueprint records intent, not a running process." },
      { label: "COMPARED", title: "Spot the missing piece", command: "openport apply web", rows: [{ name: "frontend", detail: ":3000 · running", signal: "active" }, { name: "database", detail: ":5432 · running", signal: "active" }, { name: "worker", detail: ":9000 · not running", signal: "alert" }], output: "The difference is visible immediately." },
      { label: "YOUR CALL", title: "A report, never a surprise action", command: "review report  →  choose next step", rows: [{ name: "frontend", detail: "leave running", signal: "active" }, { name: "database", detail: "leave running", signal: "protected" }, { name: "worker", detail: "start it yourself", signal: "alert" }], output: "Applying a blueprint does not start or stop services." },
    ],
  },
] satisfies {
  slug: string; number: string; category: string; time: string; title: string; excerpt: string; lead: string; promise: string;
  command: string; commandNote: string; outcome: string; steps: { title: string; body: string }[]; frames: Frame[]
}[]

export type Guide = (typeof guides)[number]
