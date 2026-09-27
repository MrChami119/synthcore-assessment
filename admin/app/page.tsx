import { auth } from "@clerk/nextjs/server";
// import { UserButton } from "@clerk/nextjs";
import StatusSelect from "./StatusSelect";

type Enquiry = {
  id: number;
  name: string;
  email: string;
  company: string;
  message: string;
  status: string;
  created_at: string;
};

async function getEnquiries(): Promise<Enquiry[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/enquiries`, {
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch enquiries: ${res.status} ${res.statusText} — ${body}`,
    );
  }

  return res.json();
}

export default async function DashboardPage() {
  await auth.protect();

  const enquiries = await getEnquiries();

  return (
    <main className="mx-auto max-w-4xl px-5 py-16">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold">Enquiries</h1>
          <p className="mt-2 text-zinc-600">{enquiries.length} total</p>
        </div>
        {/* <UserButton afterSignOutUrl="/sign-in" /> */}
      </div>

      <div className="mt-8 flex flex-col gap-4">
        {enquiries.map((enquiry) => (
          <div
            key={enquiry.id}
            className="rounded-lg border border-zinc-300 p-4 dark:border-zinc-700"
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="font-medium">{enquiry.name}</p>
                <p className="text-sm text-zinc-500">{enquiry.email}</p>
                {enquiry.company && (
                  <p className="text-sm text-zinc-500">{enquiry.company}</p>
                )}
              </div>
              <StatusSelect
                enquiryId={enquiry.id}
                currentStatus={enquiry.status}
              />
            </div>
            <p className="mt-3 text-sm">{enquiry.message}</p>
            <p className="mt-3 text-xs text-zinc-400">
              {new Date(enquiry.created_at).toLocaleString()}
            </p>
          </div>
        ))}
      </div>
    </main>
  );
}
