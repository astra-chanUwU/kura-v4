export type CommentSafetyEvent = {
  id: string;
  comment_id: string;
  post_id: string;
  author_username: string;
  action: "blocked" | "flagged";
  categories: string[];
  body: string;
  status: "open" | "dismissed" | "removed";
  created_at: string;
  resolved_at: string;
  resolver_id: string;
  resolver_username: string;
  decisions: CommentSafetyDecision[];
};

export type CommentSafetyDecision = {
  actor_id: string;
  actor_username: string;
  action: "dismissed" | "removed" | "reopened" | "restored";
  created_at: string;
};

export type CommentSafetyPage = {
  total: number;
  events: CommentSafetyEvent[];
};

export type CommentSafetyResolver = {
  id: string;
  username: string;
  decisions: number;
  last_status: "dismissed" | "removed" | "reopened" | "restored";
  last_action: "blocked" | "flagged";
  last_categories: string[];
  last_body: string;
  last_at: string;
};

export type CommentSafetyResolvers = {
  resolvers: CommentSafetyResolver[];
};

export type CommentSafetyQuery = {
  status: "open" | "dismissed" | "removed" | "any";
  action: "" | "blocked" | "flagged";
  category: string;
  q: string;
  resolver: string;
  sort: "newest" | "oldest" | "decided";
  limit: number;
  offset: number;
};
