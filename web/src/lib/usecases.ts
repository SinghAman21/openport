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
    problem: "A dev server cannot bind :3000, but the error does not identify the listener you would be stopping.",
    oldApproach: { title: "Trace the PID by hand (macOS/Linux)", command: "lsof -nP -iTCP:3000 -sTCP:LISTEN", body: "Read the PID and process name, then decide whether to send that specific process SIGTERM. Avoid a blanket kill of all Node processes." },
    newApproach: { title: "Inspect before stopping", body: "Run openport check 3000; if it is your stale server, select :3000 in the Ports view and press x, then confirm the kill." },
    command: "openport check 3000", commandNote: "Inspect the port before you touch the process.",
    outcome: "One conflict resolved. Your database, cache and other projects stay exactly where they were.",
    steps: [
      { title: "Locate the owner", story: "It’s 9:02. The frontend is ready; npm run dev isn’t. Port 3000 has a ghost in it—except this ghost has a process name and a PID.", body: "Run openport check 3000 or open the Ports view. You'll see the owning process, PID and bind address. A port number alone is not enough reason to stop a process." },
      { title: "Make the call", story: "No sweeping up every Node process. One check shows PID 5678 is yesterday’s dev server. Your database is on 5432, minding its own business.", body: "If the listener is a stale dev server, select it in the Ports view and press x. OpenPort shows a confirmation before it stops anything. If it belongs to another project, leave it alone." },
      { title: "Start clean", story: "You confirm the one stale listener. Port 3000 clears; the database stays protected. Restart the frontend and get back to the thing you were actually building.", body: "With the stale process gone, port 3000 is available to your app. Protect important listeners like your database so a future cleanup cannot catch them by mistake." },
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
    problem: "Several projects use the same generic process names, making 3000, 5432 and 6379 hard to place at a glance.",
    oldApproach: { title: "Keep a manual port map (Linux)", command: "ss -ltnp", body: "Compare listening ports and PIDs against your own notes or terminal tabs each time a repo is opened." },
    newApproach: { title: "Name the stack", body: "Run openport --view aliases to name ports in the interface; put aliases and expected_ports in .openport.yml to share the map with the repo." },
    command: "openport --view aliases", commandNote: "Start naming the services you already run.",
    outcome: "Your stack reads like a map instead of a list of numbers. Project config makes that map portable with the repo.",
    steps: [
      { title: "Name what you know", story: "Monday morning. Three repos open. Every terminal says node, postgres, redis—and none of them say which project they belong to.", body: "Open the Aliases view and name port 3000 frontend, 5432 database and 6379 cache. The names appear beside live listeners in the Ports view." },
      { title: "Put the map in the repo", story: "Give those numbers names once. Now the port table reads like the stack you recognize, not a memory test you have to repeat after every restart.", body: "Create .openport.yml with aliases and expected_ports. OpenPort discovers the file from your working directory or a parent, even when you launch from a subdirectory." },
      { title: "Set a guardrail", story: "Add the project map to .openport.yml and protect 5432. Tomorrow, and on the next branch, your database is still clearly the database—and safer from a rushed cleanup.", body: "Add 5432 to the project's protected list. The ordinary kill path will refuse it; overriding protection requires extra confirmation in the interface." },
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
    problem: "A second worker needs a local port, and the next number might already be listening or reserved.",
    oldApproach: { title: "Check candidates one at a time (macOS/Linux)", command: "lsof -nP -iTCP:4000 -sTCP:LISTEN", body: "Repeat for 4001, 4002 and so on. No listener in this listing still does not prove the port can be bound or is unreserved." },
    newApproach: { title: "Ask for a bindable candidate", body: "Run openport free --near 4000. Use the returned port in your service configuration; reserve it in the interface if it should not be suggested again." },
    command: "openport free --near 4000", commandNote: "Ask for an available port near your usual range.",
    outcome: "The output is a bare port number. Use it in your local app or capture it in a script; reserve it if that service should keep it.",
    steps: [
      { title: "Ask near your range", story: "A worker needs a local endpoint. You try 4000; it’s busy. 4001 is reserved. Time to stop playing port roulette.", body: "Run openport free --near 4000. OpenPort skips ports that are listening, reserved, or protected, then tests whether the candidate can actually be bound." },
      { title: "Use the result", story: "OpenPort checks the candidates against the machine—not just the list—and prints a bare number. In this example, 4002 passes the bind check.", body: "The command prints a bare number on stdout, which is easy to pass into your own app configuration or capture in a script. The port shown here is an illustration, not a promise about your machine." },
      { title: "Keep the slot", story: "Wire the suggested port into your app. If this worker should keep its number, reserve it. If it changes later, nothing has silently claimed that slot for good.", body: "If that port belongs to a specific service, reserve it in OpenPort. Future free-port suggestions will not hand it to something else." },
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
    problem: "The app is reachable but a worker is absent; a list of listeners does not say what this project expected to be running.",
    oldApproach: { title: "Compare the stack by hand (Linux)", command: "ss -ltnp", body: "Check each expected port against a README or your own notes, then investigate any missing or unexpected listener yourself." },
    newApproach: { title: "Compare intent with reality", body: "Create a blueprint in the Blueprints view, then run openport apply web to get a read-only report of missing or displaced services." },
    command: "openport apply web", commandNote: "After creating a blueprint named web in the UI.",
    outcome: "Applying is read-only: it tells you what's missing or displaced and suggests a next step. It never starts or kills a process for you.",
    steps: [
      { title: "Describe the expected environment", story: "It’s almost stand-up. The app loads, but a background worker is missing. You know it should be there—you just can’t remember which port this project expects.", body: "In the Blueprints view, create a blueprint called web with the services, ports and health probes your project expects." },
      { title: "Compare against what is running", story: "A blueprint puts expectation next to reality. Frontend: up. Database: up. Worker: not running. No archaeology through terminal tabs required.", body: "Apply the blueprint in the UI or run openport apply web. OpenPort reports which services are up, absent, or blocked by a different listener." },
      { title: "Fix with intention", story: "You decide what to do with the missing worker. The report points; it never presses the button for you. That’s useful when the wrong “fix” could take down someone else’s service.", body: "The report is a plan, not an automation run. Choose what to start, move or leave alone yourself. OpenPort won't kill or launch anything just because a blueprint says so." },
    ],
    frames: [
      { label: "EXPECTED", title: "What this project needs", command: "blueprint: web", rows: [{ name: "frontend", detail: ":3000 · expected", signal: "active" }, { name: "database", detail: ":5432 · expected", signal: "active" }, { name: "worker", detail: ":9000 · expected", signal: "muted" }], output: "A blueprint records intent, not a running process." },
      { label: "COMPARED", title: "Spot the missing piece", command: "openport apply web", rows: [{ name: "frontend", detail: ":3000 · running", signal: "active" }, { name: "database", detail: ":5432 · running", signal: "active" }, { name: "worker", detail: ":9000 · not running", signal: "alert" }], output: "The difference is visible immediately." },
      { label: "YOUR CALL", title: "A report, never a surprise action", command: "review report  →  choose next step", rows: [{ name: "frontend", detail: "leave running", signal: "active" }, { name: "database", detail: "leave running", signal: "protected" }, { name: "worker", detail: "start it yourself", signal: "alert" }], output: "Applying a blueprint does not start or stop services." },
    ],
  },
  {
    slug: "watch-mode", number: "05", category: "OBSERVABILITY", time: "2 MIN READ", title: "See a restart as it happens.",
    excerpt: "Watch a listener disappear and return without rerunning a scan or losing your place.",
    lead: "The dev server restarts, but your browser still fails. Did it come back on the same port? Watch the listening table while you restart it.",
    promise: "Follow the change, not a pile of snapshots.",
    problem: "A server restarts quickly, and one static snapshot cannot show whether its listener returned or moved.",
    oldApproach: { title: "Poll the listener list (Linux)", command: "watch -n 1 'ss -ltnp'", body: "Leave a second terminal polling ports and compare the output between refreshes; correlate PIDs manually when processes restart." },
    newApproach: { title: "Watch in the Ports view", body: "Run openport, press w in the Ports view, then restart your service in another terminal. Watch the port delta and selected process; press w again to return to normal refresh." },
    command: "openport", commandNote: "Press w in the Ports view to toggle faster watch mode.",
    outcome: "The listener change is visible in the interactive table; you still decide what, if anything, to fix.",
    steps: [
      { title: "Start watching", story: "A save triggers a restart. The browser spins, and you need to know whether :3000 came back at all.", body: "Launch openport, stay in Ports, and press w. The watching indicator shows faster refresh is on; the regular list is live too, just at a slower cadence." },
      { title: "Restart your service", story: "The old listener drops away. Moments later, a new PID appears on :3000—or the port stays empty.", body: "Restart the server in its own terminal. Watch the table's added/removed delta and the selected process as rows change; no repeated scan command is needed." },
      { title: "Check the result", story: "The new listener is there. The failed page load has another cause; you can stop chasing a port conflict.", body: "Select :3000 to inspect the current owner. If nothing is listening, troubleshoot the server's own startup logs. Press w to toggle watch mode off." },
    ],
    frames: [
      { label: "RESTARTING", title: "The old listener disappears", command: "Ports view · watch on", rows: [{ name: ":3000", detail: "server restarting", signal: "alert" }, { name: ":5432", detail: "database · listening", signal: "muted" }], output: "A snapshot could miss this transition." },
      { label: "REFRESHED", title: "A new PID arrives", command: "openport  →  w", rows: [{ name: ":3000", detail: "node · new PID 6721", signal: "active" }, { name: ":5432", detail: "database · listening", signal: "muted" }], output: "The live table tracks additions and removals." },
      { label: "VERIFIED", title: "Inspect what came back", command: "Ports view · select :3000", rows: [{ name: ":3000", detail: "node · listening", signal: "active" }, { name: ":5432", detail: "database · unchanged", signal: "muted" }], output: "Example PIDs only. Watch mode never restarts the server." },
    ],
  },
  {
    slug: "groups", number: "06", category: "WORKFLOW", time: "3 MIN READ", title: "Close the right stack together.",
    excerpt: "Check a group of listeners before shutting down your local stack with one confirmed action.",
    lead: "The frontend, API and worker are still running after you finish work. You want to check the stack first, then shut it down without hunting three PIDs.",
    promise: "Treat the stack as a unit, without skipping the safety check.",
    problem: "A multi-service dev stack spans several ports, and stopping each process by hand is easy to get wrong.",
    oldApproach: { title: "Check and stop managed services (PM2)", command: "pm2 list; pm2 stop frontend api worker", body: "If those processes are managed by PM2, inspect their status and stop them by name. Other terminals or unmanaged processes still need separate handling." },
    newApproach: { title: "Group ports in OpenPort", body: "Run openport --view aliases, switch to Groups, add your stack's ports, press H to health-check the group, then x to review and confirm shutdown. Protected ports remain guarded." },
    command: "openport --view aliases", commandNote: "Switch to Groups in this view; use H for health and x for confirmed shutdown.",
    outcome: "One group gives the stack a shared health check and shutdown workflow, while protected members cannot be silently killed.",
    steps: [
      { title: "Bundle the listeners", story: "The frontend is on :3000, API on :8080 and worker on :9000. Three terminals are open; their PIDs are not worth memorizing.", body: "Open the Aliases view, press tab to switch to Groups, then n to create a group for ports 3000, 8080 and 9000. Groups organize existing listeners; they do not launch them." },
      { title: "Check before closing", story: "The worker might already have stopped. Check the group's health before deciding what still needs attention.", body: "Select the group and press H to run a group health check. Review the results for its members rather than assuming every service is healthy." },
      { title: "Confirm the shutdown", story: "Work is done. The group gives you one place to act, but not an unchecked off switch.", body: "Select the group and press x. Review the confirmation before shutdown; protected ports are still subject to protection rules. Leave the stack running if anything looks wrong." },
    ],
    frames: [
      { label: "SCATTERED", title: "Three services to track", command: "PM2 or terminal tabs · example", rows: [{ name: ":3000", detail: "frontend", signal: "muted" }, { name: ":8080", detail: "api", signal: "muted" }, { name: ":9000", detail: "worker", signal: "muted" }], output: "Separate processes do not come with a shared check." },
      { label: "GROUPED", title: "Check dev_stack together", command: "Groups view · select dev_stack · H", rows: [{ name: "frontend", detail: ":3000 · health checked", signal: "active" }, { name: "api", detail: ":8080 · health checked", signal: "active" }, { name: "worker", detail: ":9000 · review result", signal: "alert" }], output: "Example results: investigate an unhealthy member before stopping anything." },
      { label: "CONFIRMED", title: "Review before shutdown", command: "Groups view · select dev_stack · x", rows: [{ name: "frontend", detail: ":3000 · group member", signal: "active" }, { name: "api", detail: ":8080 · group member", signal: "active" }, { name: "worker", detail: ":9000 · group member", signal: "muted" }], output: "Shutdown requires confirmation and respects protected ports." },
    ],
  },
  {
    slug: "protected-ports", number: "07", category: "SAFETY", time: "3 MIN READ", title: "Keep the ports you cannot afford to kill.",
    excerpt: "Protect important listeners, and make overrides deliberate instead of accidental.",
    lead: "A cleanup is tempting when a port looks busy. But SSH or your local database should not disappear because you picked the wrong row.",
    promise: "Build a guardrail before the next rushed cleanup.",
    problem: "A bulk or mistaken kill could stop a critical listener, including a service needed to keep the machine accessible.",
    oldApproach: { title: "Inspect each target manually (macOS/Linux)", command: "lsof -nP -iTCP:22 -sTCP:LISTEN", body: "Identify the owner of each port before any kill or range cleanup. A shell kill has no project-specific protected list to catch a bad choice." },
    newApproach: { title: "Use explicit protection", body: "Run openport --view config and open Protected to manage your own ports, ranges or process names. The ordinary kill path refuses protected targets; overrides are only available in the UI and require two confirmations." },
    command: "openport --view config", commandNote: "Open Protected to review and manage your guardrails.",
    outcome: "Critical listeners stay off the ordinary kill path; a range cleanup skips protected ports and reports why.",
    steps: [
      { title: "Review the defaults", story: "The cleanup list includes :22. Stopping SSH could end the session you are using to reach this machine.", body: "Open Config → Protected. OpenPort ships with a never-kill list for services such as SSH and DNS; inspect it before cleaning up a range." },
      { title: "Add your own guardrail", story: "Your database on :5432 matters too, even though it is not a system service.", body: "In the Protected tab, protect the database port, a range, or its process name. Project config can also list protected ports in .openport.yml." },
      { title: "Choose deliberately", story: "A later cleanup touches :5432. OpenPort refuses the ordinary kill instead of turning a typo into lost work.", body: "Read the refusal reason and leave the service running. Only if you truly intend to stop a protected listener, use the interface's override flow and complete both confirmations; CLI kills cannot override protection." },
    ],
    frames: [
      { label: "AT RISK", title: "A rushed cleanup", command: "manual port review · example", rows: [{ name: ":22", detail: "sshd · needed", signal: "alert" }, { name: ":5432", detail: "postgres · needed", signal: "alert" }, { name: ":3000", detail: "stale dev server", signal: "muted" }], output: "A PID is not enough context for a safe kill." },
      { label: "GUARDED", title: "Mark what must survive", command: "openport --view config", rows: [{ name: ":22", detail: "system protection", signal: "protected" }, { name: ":5432", detail: "project protection", signal: "protected" }, { name: ":3000", detail: "unprotected", signal: "muted" }], output: "Manage protections in Config → Protected." },
      { label: "SAFE", title: "The ordinary path refuses", command: "Ports view · select :5432 · x", rows: [{ name: ":22", detail: "remains protected", signal: "protected" }, { name: ":5432", detail: "kill refused", signal: "protected" }, { name: ":3000", detail: "review separately", signal: "active" }], output: "Overrides require two UI confirmations; the CLI cannot override." },
    ],
  },
] satisfies {
  slug: string; number: string; category: string; time: string; title: string; excerpt: string; lead: string; promise: string;
  problem: string; oldApproach: { title: string; command: string; body: string }; newApproach: { title: string; body: string };
  command: string; commandNote: string; outcome: string; steps: { title: string; story: string; body: string }[]; frames: Frame[]
}[]

export type Guide = (typeof guides)[number]
