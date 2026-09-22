import { redirect } from "next/navigation";

// The template's landing route. Home is "/" now.
export default function DashboardPage() {
  redirect("/");
}
