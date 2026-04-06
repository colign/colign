import { createClient } from "@connectrpc/connect";
import { WikiService } from "@/gen/proto/wiki/v1/wiki_pb";
import { transport } from "./connect";

export const wikiClient = createClient(WikiService, transport);
