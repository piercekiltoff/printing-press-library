export async function searchCommand(args: string[]): Promise<number> {
  const query = args.join(" ").trim();
  if (!query) {
    console.error("Usage: pp search <query>");
    return 1;
  }

  console.log(`search scaffold ready for ${query}; implementation lands in U5.`);
  return 0;
}
