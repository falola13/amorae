"use client";

import { Icon } from "@/components/icons";
import { PickRow } from "@/components/ui/pick-row";
import {
  atTimeReminder,
  eventReminderLabel,
  MAX_REMINDERS,
  reminderTime,
  REMINDER_OPTIONS,
} from "@/features/together/events";

/** The "At a set time…" choice; it stands for any "at HH:MM" value. */
const SET_TIME = "__at__";
/** What a freshly chosen set time starts as, before it's changed. */
const DEFAULT_TIME = "09:00";

const PHRASES = REMINDER_OPTIONS.filter((o) => o.value !== "");

/** The row's choice: its phrase, or SET_TIME for an "at HH:MM". */
const choiceOf = (value: string) => (reminderTime(value) ? SET_TIME : value);

/**
 * Up to three reminders for one event: a row each, with a way to remove it, and a link to add another.
 * Each row offers only the phrases the other rows haven't taken. Without `onChange` it just shows them.
 */
export function RemindersEditor({
  value,
  onChange,
  error,
}: {
  value: readonly string[];
  onChange?: (next: string[]) => void;
  error?: string;
}) {
  const taken = new Set(value.filter((v) => !reminderTime(v)));
  const unused = PHRASES.filter((o) => !taken.has(o.value));

  const optionsFor = (own: string) => [
    ...PHRASES.filter((o) => o.value === own || !taken.has(o.value)),
    { value: SET_TIME, label: "At a set time…" },
  ];

  const pick = (choice: string): string =>
    choice === SET_TIME ? atTimeReminder(DEFAULT_TIME) : choice;
  const replaceAt = (i: number, next: string) =>
    onChange?.(value.map((v, j) => (j === i ? next : v)));

  if (value.length === 0) {
    return (
      <div>
        <PickRow
          icon="bell"
          label="Reminder"
          value=""
          options={[{ value: "", label: "None" }, ...optionsFor("")]}
          onChange={onChange ? (c) => c && onChange([pick(c)]) : undefined}
          placeholder="None"
          last
        />
        {error ? <ErrorLine text={error} /> : null}
      </div>
    );
  }

  return (
    <div>
      {value.map((v, i) => {
        const time = reminderTime(v);
        return (
          <div key={i}>
            <div className="flex items-center">
              <div className="min-w-0 grow">
                <PickRow
                  icon="bell"
                  label={value.length > 1 ? `Reminder ${i + 1}` : "Reminder"}
                  value={choiceOf(v)}
                  display={time ? "At a set time" : undefined}
                  options={optionsFor(choiceOf(v))}
                  onChange={onChange ? (c) => replaceAt(i, pick(c)) : undefined}
                  placeholder="None"
                  last={!time && i === value.length - 1 && !onChange}
                />
              </div>
              {onChange ? (
                <button
                  type="button"
                  aria-label={`Remove ${eventReminderLabel(v).toLowerCase()} reminder`}
                  onClick={() => onChange(value.filter((_, j) => j !== i))}
                  className="press -mr-2 flex h-11 w-11 shrink-0 items-center justify-center text-stone"
                >
                  <Icon name="x" size={18} />
                </button>
              ) : null}
            </div>
            {time ? (
              <PickRow
                icon="clock"
                label="At"
                value={time}
                type="time"
                onChange={onChange ? (t) => t && replaceAt(i, atTimeReminder(t)) : undefined}
                placeholder="Pick a time"
                last={i === value.length - 1 && !onChange}
              />
            ) : null}
          </div>
        );
      })}
      {onChange && value.length < MAX_REMINDERS ? (
        <button
          type="button"
          onClick={() =>
            onChange([...value, unused[0] ? unused[0].value : atTimeReminder(DEFAULT_TIME)])
          }
          className="press flex h-11 items-center gap-2 text-[15px] font-semibold text-plum"
        >
          <Icon name="plus" size={16} />
          Add another reminder
        </button>
      ) : null}
      {error ? <ErrorLine text={error} /> : null}
    </div>
  );
}

function ErrorLine({ text }: { text: string }) {
  return (
    <div role="alert" className="mt-1 text-[13px] text-red">
      {text}
    </div>
  );
}
