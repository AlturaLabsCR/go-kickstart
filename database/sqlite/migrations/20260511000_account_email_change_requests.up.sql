CREATE TABLE IF NOT EXISTS account_email_change_requests (
  sub INTEGER PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  otp INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,

  CONSTRAINT account_email_change_requests_email_len CHECK (length(email) <= 255),
  CONSTRAINT account_email_change_requests_otp_range CHECK (otp >= 0 AND otp <= 999999)
);
