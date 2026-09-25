"use client";

import { Icon } from "@/components/icons";
import { useCouple } from "../hooks";

/** Said wherever a write pings the partner, so nobody finds out afterwards
 *  that a private-feeling line was announced. Nothing when there's no partner. */
export function PartnerNotice() {
  const partner = useCouple().data?.partner?.display_name;
  if (!partner) return null;
  return (
    <div className="flex items-center gap-2 text-support text-stone">
      <Icon name="bell" size={16} />
      {partner} will get one quiet notification.
    </div>
  );
}
