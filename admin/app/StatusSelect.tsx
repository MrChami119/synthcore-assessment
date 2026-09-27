"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

const STATUSES = ["New", "In Progress", "Closed"];

export default function StatusSelect({
  enquiryId,
  currentStatus,
}: {
  enquiryId: number;
  currentStatus: string;
}) {
  const [status, setStatus] = useState(currentStatus);
  const [updating, setUpdating] = useState(false);
  const router = useRouter();

  async function handleChange(newStatus: string) {
    setUpdating(true);
    const previous = status;
    setStatus(newStatus); // optimistic update

    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/enquiries/${enquiryId}/status`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status: newStatus }),
        },
      );

      if (!res.ok) {
        throw new Error("Failed to update status");
      }

      router.refresh(); // re-fetch server component data to stay in sync
    } catch (err) {
      setStatus(previous); // roll back on failure
      alert("Failed to update status. Please try again.");
    } finally {
      setUpdating(false);
    }
  }

  return (
    <select
      value={status}
      disabled={updating}
      onChange={(e) => handleChange(e.target.value)}
      className="rounded-full border border-zinc-300 bg-zinc-100 px-3 py-1 text-xs font-medium dark:border-zinc-700 dark:bg-zinc-800"
    >
      {STATUSES.map((s) => (
        <option key={s} value={s}>
          {s}
        </option>
      ))}
    </select>
  );
}
