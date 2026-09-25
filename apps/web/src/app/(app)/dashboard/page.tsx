import { redirect } from "next/navigation";

// Not dead: the first manifest opened the app at /dashboard, and iOS keeps the
// start URL an app was installed with. Deleting this would open those to a 404.
export default function DashboardPage() {
  redirect("/");
}
