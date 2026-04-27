import test from "node:test";
import assert from "node:assert/strict";
import { skillSource, skillsAddArgs } from "../src/skill.js";

test("skillSource points at the cli-skills namespace", () => {
  assert.equal(
    skillSource("pp-espn"),
    "mvanhorn/printing-press-library/cli-skills/pp-espn",
  );
});

test("skillsAddArgs uses unattended skills add with optional agents", () => {
  assert.deepEqual(skillsAddArgs("pp-espn", { agents: ["claude-code", "codex"] }), [
    "-y",
    "skills@latest",
    "add",
    "mvanhorn/printing-press-library/cli-skills/pp-espn",
    "-g",
    "-y",
    "-a",
    "claude-code",
    "-a",
    "codex",
  ]);
});
