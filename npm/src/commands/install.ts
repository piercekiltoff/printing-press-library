export async function installCommand(args: string[]): Promise<number> {
  const name = args[0];
  if (!name) {
    console.error("Usage: pp install <name>");
    return 1;
  }

  console.log(`install scaffold ready for ${name}; implementation lands in U4.`);
  return 0;
}
