export function Scripture({
  reference,
  verse,
  faint,
}: {
  reference: string;
  verse?: string;
  faint?: boolean;
}) {
  return (
    <div
      className={`flex flex-col gap-1.5 border-t pt-5 ${faint ? "border-faint" : "border-line"}`}
    >
      <div className="text-support font-bold tracking-[0.04em] text-plum">{reference}</div>
      {verse ? (
        <div className="font-text text-[17px] leading-[1.55] italic text-stone">{verse}</div>
      ) : null}
    </div>
  );
}
