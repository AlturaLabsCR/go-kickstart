CREATE TABLE IF NOT EXISTS account_login_requests (
  email VARCHAR(255) PRIMARY KEY NOT NULL,
  otp BIGINT NOT NULL,
  expires_at BIGINT NOT NULL,

  CONSTRAINT account_login_requests_email_len CHECK (length(email) <= 255),
  CONSTRAINT account_login_requests_otp_range CHECK (otp >= 0 AND otp <= 999999)
);
