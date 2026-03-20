import { createClient } from "@connectrpc/connect";
import { AcceptanceCriteriaService } from "@/gen/proto/acceptance/v1/acceptance_pb";
import { transport } from "./connect";

export const acceptanceClient = createClient(AcceptanceCriteriaService, transport);
