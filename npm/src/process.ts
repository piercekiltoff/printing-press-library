import { execFile } from "node:child_process";

export interface RunResult {
  code: number;
  stdout: string;
  stderr: string;
}

export interface RunOptions {
  env?: NodeJS.ProcessEnv;
}

export type Runner = (command: string, args: string[], options?: RunOptions) => Promise<RunResult>;

export const execFileRunner: Runner = (command, args, options = {}) => {
  return new Promise((resolve) => {
    execFile(
      command,
      args,
      {
        env: options.env ? { ...process.env, ...options.env } : process.env,
      },
      (error, stdout, stderr) => {
        if (error && "code" in error && error.code === "ENOENT") {
          resolve({ code: 127, stdout, stderr });
          return;
        }

        resolve({
          code: typeof error?.code === "number" ? error.code : 0,
          stdout,
          stderr,
        });
      },
    );
  });
};
