/** A milestone date recurs yearly, so "next" means today if it hasn't
 * happened yet this year, otherwise the same day next year. Shared by the
 * hub (soonest upcoming date) and the milestones list (grouping). */
export const nextOccurrence = (date: string, todayIso: string) => {
  if (date >= todayIso) return date;
  const [, m, d] = date.split("-");
  const y = Number(todayIso.slice(0, 4));
  const thisYear = `${y}-${m}-${d}`;
  return thisYear >= todayIso ? thisYear : `${y + 1}-${m}-${d}`;
};
