# Agent Guide

Quick, repo-specific instructions for humans and AI agents working in this codebase.

## Index

- [TypeScript and code level instructions](#typescript-and-code-level-instructions)
  - [TypeScript imports](#typescript-imports)
  - [Next server hanging](#next-server-hanging)
  - [Convex dev data / seeding (quick workflow)](#convex-dev-data--seeding-quick-workflow)
  - [Run convex functions from the command line](#run-convex-functions-from-the-command-line)
  - [Run convex and next separately as needed](#run-convex-and-next-separately-as-needed)
  - [Server rendered component with convex](#server-rendered-component-with-convex)
- [Playwright CLI](#playwright-cli)
  - [Quick Start](#playwright-cli-quick-start)
  - [Available Variables](#playwright-cli-available-variables)
  - [Examples](#playwright-cli-examples)

# typescript and code level instructions

## typescript imports

Use @/ to refer to things from the root directory.

## next server hanging

It's usually the nextjs server that hangs.
If accessing the site fails, you may need to kill the next-server manually and run the `npm run dev:next` script again.

Because this happens so much - simply terminate the `npm run dev` process when you're done using it.

## Convex dev data / seeding (quick workflow)

When doing a task, if you need to load data into Convex, **keep it simple**:

- Create a **seed mutation** (e.g. `seedXxx`) that is **idempotent**.
- Call it on **page load** (client-side) for the relevant page/screen.
- Inside the mutation: **check if data already exists**, and **only seed if no data exists** (otherwise no-op).

This avoids separate seed scripts and keeps setup “open the page and it works”.

## Run convex functions from the command line

> npx convex run --help
Usage: convex run [options] <functionName> [args]

Run a function (query, mutation, or action) on your deployment

Arguments:
  functionName                        identifier of the function to run, like `listMessages` or `dir/file:myFunction`
  args                                JSON-formatted arguments object to pass to the function.

Options:
  --push                              Push code to deployment before running the function.
  --identity <identity>               JSON-formatted UserIdentity object, e.g. '{ name: "John", address: "0x123" }'

## Run convex and next separately as needed

```bash
npm run dev:next
npm run dev:convex
```

## Server rendered component with convex

Here's how you do server rendered component with convex:

```
import { convexAuthNextjsToken } from "@convex-dev/auth/nextjs/server";
import { api } from "@/convex/_generated/api";
import { fetchMutation, fetchQuery } from "convex/nextjs";
import { revalidatePath } from "next/cache";
 
export default async function PureServerPage() {
  const tasks = await fetchQuery(api.tasks.list, { list: "default" });
  async function createTask(formData: FormData) {
    "use server";
 
    await fetchMutation(
      api.tasks.create,
      {
        text: formData.get("text") as string,
      },
      { token: await convexAuthNextjsToken() },
    );
    revalidatePath("/example");
  }
  // render tasks and task creation form
  return <form action={createTask}>...</form>;
}
```

The same applies for `convex.query()` on the server





# playwright-cli

Use the playwright-cli instead of opening any other browser you've been told to use.
It acts as a browser but is faster than the other one you were told to use - so use it!
ALWAYS open the browser in HEADLESS mode (not --headed)
Make sure to close your page when you're done with it, but leave playwright-cli as started (don't stop it).
Always save screenshots to /tmp


## playwright-cli Quick Start

FIRST! Install the playwright cli:

`npm install -g @mj1618/playwright-cli`

```bash
# Start the browser server (headless by default)
playwright-cli start

# Create a new page and get its pageId
playwright-cli new-page
# Returns: abc12345

# Run commands against the browser using regular playwright js calls (pageId required)
playwright-cli -e "await page.goto('https://example.com')" --page abc12345
playwright-cli -e "await page.title()" --page abc12345

# Close the page when done
playwright-cli close-page abc12345

# Stop the server when completely done
playwright-cli stop
```

## playwright-cli Available Variables

When executing code, these variables are in scope:

- `page` - the current [Page](https://playwright.dev/docs/api/class-page) (for the specified pageId)
- `browser` - the [Browser](https://playwright.dev/docs/api/class-browser) instance
- `context` - the [BrowserContext](https://playwright.dev/docs/api/class-browsercontext)

## playwright-cli Examples

```bash
# Create a page first
PAGE_ID=$(playwright-cli new-page)

# Navigate and interact
playwright-cli -e "await page.goto('https://github.com')" --page $PAGE_ID
playwright-cli -e "await page.click('a[href=\"/login\"]')" --page $PAGE_ID
playwright-cli -e "await page.fill('#login_field', 'username')" --page $PAGE_ID

# Get page info
playwright-cli -e "await page.title()" --page $PAGE_ID
playwright-cli -e "await page.url()" --page $PAGE_ID

# Screenshots
playwright-cli -e "await page.screenshot({ path: 'screenshot.png' })" --page $PAGE_ID

# Evaluate in browser context
playwright-cli -e "await page.evaluate(() => document.body.innerText)" --page $PAGE_ID

# List all active pages
playwright-cli list-pages

# Close the page when done
playwright-cli close-page $PAGE_ID
```
