package email

import "fmt"

// VerifyEmail builds a verification email.
func VerifyEmail(name, verifyURL string) SendParams {
	return SendParams{
		Subject: "Confirmă adresa de email — EtiketAI",
		HTML: fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:24px">
<h2 style="color:#1a1a2e">Bun venit la EtiketAI, %s!</h2>
<p>Confirmă adresa de email apăsând butonul de mai jos:</p>
<a href="%s" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600;margin:16px 0">Confirmă email</a>
<p style="color:#666;font-size:13px">Linkul expiră în 24 de ore. Dacă nu ai creat un cont, ignoră acest mesaj.</p>
<hr style="border:none;border-top:1px solid #eee;margin:24px 0">
<p style="color:#999;font-size:12px">EtiketAI · Conformitate etichete produs</p>
</body></html>`, name, verifyURL),
		Text: fmt.Sprintf("Bun venit la EtiketAI, %s!\n\nConfirmă adresa de email: %s\n\nLinkul expiră în 24 de ore.", name, verifyURL),
	}
}

// ResetPassword builds a password reset email.
func ResetPassword(name, resetURL string) SendParams {
	return SendParams{
		Subject: "Resetare parolă — EtiketAI",
		HTML: fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:24px">
<h2 style="color:#1a1a2e">Resetare parolă</h2>
<p>Salut, %s. Am primit o cerere de resetare a parolei pentru contul tău EtiketAI.</p>
<a href="%s" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600;margin:16px 0">Resetează parola</a>
<p style="color:#666;font-size:13px">Linkul expiră în 1 oră. Dacă nu ai solicitat resetarea, ignoră acest mesaj.</p>
</body></html>`, name, resetURL),
		Text: fmt.Sprintf("Salut, %s.\n\nResetează parola: %s\n\nLinkul expiră în 1 oră.", name, resetURL),
	}
}

// WorkspaceInvite builds a workspace invitation email.
func WorkspaceInvite(inviterName, workspaceName, inviteURL string) SendParams {
	return SendParams{
		Subject: fmt.Sprintf("Invitație în workspace-ul %s — EtiketAI", workspaceName),
		HTML: fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:24px">
<h2 style="color:#1a1a2e">Ai fost invitat!</h2>
<p><strong>%s</strong> te-a invitat să te alături workspace-ului <strong>%s</strong> pe EtiketAI.</p>
<a href="%s" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600;margin:16px 0">Acceptă invitația</a>
<p style="color:#666;font-size:13px">Invitația expiră în 48 de ore.</p>
</body></html>`, inviterName, workspaceName, inviteURL),
		Text: fmt.Sprintf("%s te-a invitat în workspace-ul %s.\n\nAcceptă: %s\n\nExpiră în 48 de ore.", inviterName, workspaceName, inviteURL),
	}
}

// SubscriptionExpiringSoon builds a subscription expiry warning email.
func SubscriptionExpiringSoon(workspaceName string, daysLeft int) SendParams {
	return SendParams{
		Subject: fmt.Sprintf("Abonamentul tău expiră în %d zile — EtiketAI", daysLeft),
		HTML: fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:24px">
<h2 style="color:#1a1a2e">Abonament în curs de expirare</h2>
<p>Abonamentul workspace-ului <strong>%s</strong> expiră în <strong>%d zile</strong>.</p>
<p>Reînnoiește-l pentru a continua să utilizezi EtiketAI fără întreruperi.</p>
<a href="https://app.etiketai.ro/billing" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600;margin:16px 0">Reînnoiește acum</a>
</body></html>`, workspaceName, daysLeft),
		Text: fmt.Sprintf("Abonamentul workspace-ului %s expiră în %d zile. Reînnoiește: https://app.etiketai.ro/billing", workspaceName, daysLeft),
	}
}
