import { createClient } from "@connectrpc/connect";
import { CommentService } from "@/gen/proto/comment/v1/comment_pb";
import { transport } from "./connect";

export const commentClient = createClient(CommentService, transport);
