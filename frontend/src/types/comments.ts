export interface Comment {
  id: string;
  post_id: string;
  author_username: string;
  author_avatar_url: string;
  body: string;
  created_at: string;
  updated_at: string;
  can_edit: boolean;
  can_delete: boolean;
}

export interface CommentPage {
  total: number;
  comments: Comment[];
}
