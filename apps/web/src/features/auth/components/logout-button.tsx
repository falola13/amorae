import { Button } from "@/components/ui/button";
import { logoutAction } from "../actions";

export function LogoutButton() {
  return (
    <form action={logoutAction}>
      <Button type="submit" variant="secondary">
        Sign out
      </Button>
    </form>
  );
}
