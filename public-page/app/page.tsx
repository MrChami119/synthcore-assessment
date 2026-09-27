"use client";

import { useState, FormEvent } from "react";

type SubmitState = "idle" | "submitting" | "success" | "error";

const fieldClassName =
  "mt-1.5 block w-full rounded-lg border border-zinc-400 bg-white px-3 py-2.5 text-base text-zinc-900 placeholder:text-zinc-400 shadow-sm outline-none transition focus:border-zinc-800 focus:ring-2 focus:ring-zinc-800/20 dark:border-zinc-500 dark:bg-zinc-900 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus:border-zinc-300 dark:focus:ring-zinc-300/20";

export default function Home() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [company, setCompany] = useState("");
  const [message, setMessage] = useState("");
  const [state, setState] = useState<SubmitState>("idle");
  const [errorMessage, setErrorMessage] = useState("");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setState("submitting");
    setErrorMessage("");

    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/enquiries`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, company, message }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Something went wrong. Please try again.");
      }

      setState("success");
      setName("");
      setEmail("");
      setCompany("");
      setMessage("");
    } catch (err) {
      setState("error");
      setErrorMessage(err instanceof Error ? err.message : "Something went wrong.");
    }
  }

  return (
    <main className="mx-auto max-w-lg px-5 py-16 font-sans">
      <h1 className="text-3xl font-semibold tracking-tight">Get in touch</h1>
      <p className="mt-2 text-zinc-600 dark:text-zinc-400">
        Fill in the form below and we&apos;ll get back to you.
      </p>

      <form onSubmit={handleSubmit} className="mt-8 flex flex-col gap-5">
        <label className="block text-sm font-medium">
          Name *
          <input
            type="text"
            required
            autoComplete="name"
            placeholder="Your full name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={fieldClassName}
          />
        </label>

        <label className="block text-sm font-medium">
          Email *
          <input
            type="email"
            required
            autoComplete="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={fieldClassName}
          />
        </label>

        <label className="block text-sm font-medium">
          Company
          <input
            type="text"
            autoComplete="organization"
            placeholder="Company name (optional)"
            value={company}
            onChange={(e) => setCompany(e.target.value)}
            className={fieldClassName}
          />
        </label>

        <label className="block text-sm font-medium">
          Message *
          <textarea
            required
            rows={5}
            placeholder="How can we help?"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            className={`${fieldClassName} resize-y`}
          />
        </label>

        <button
          type="submit"
          disabled={state === "submitting"}
          className="rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
        >
          {state === "submitting" ? "Sending..." : "Send enquiry"}
        </button>

        {state === "success" && (
          <p className="text-sm font-medium text-green-700 dark:text-green-400">
            Thanks! Your enquiry has been received.
          </p>
        )}
        {state === "error" && (
          <p className="text-sm font-medium text-red-600 dark:text-red-400">{errorMessage}</p>
        )}
      </form>
    </main>
  );
}
