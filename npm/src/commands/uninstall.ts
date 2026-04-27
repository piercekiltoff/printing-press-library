export async function uninstallCommand(args: string[]): Promise<number> {
  const name = args[0];
  if (!name) {
    console.error("Usage: pp uninstall <name>");
    return 1;
  }

  console.log(`uninstall scaffold ready for ${name}; implementation lands in U5.`);
  return 0;
}
