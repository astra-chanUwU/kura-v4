// Keep these wire contracts in sync with internal/dto. They are handwritten.
export interface Site {
  name: string;
  description: string;
}
export interface ApiErrorBody {
  code: string;
  message: string;
  request_id: string;
}
