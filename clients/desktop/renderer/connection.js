'use strict'
const $ = (id) => document.getElementById(id)
window.helixShell.getConnection().then((c) => {
  $('url').value = c.serverUrl; $('user').value = c.username
  $('pw').placeholder = c.hasPassword ? 'saved (leave blank to keep for this server)' : ''
  $('note').textContent = c.encryptionAvailable ? 'The password is stored encrypted with the OS keystore.' : 'No OS keystore is available: the password is kept for this session only and is never written to disk.'
})
$('f').addEventListener('submit', async (e) => {
  e.preventDefault()
  const r = await window.helixShell.saveConnection({ serverUrl: $('url').value, username: $('user').value, password: $('pw').value })
  $('pw').value = ''
  $('msg').textContent = r.ok ? 'Saved.' : r.error
})
$('forget').addEventListener('click', async () => {
  const r = await window.helixShell.forgetPassword()
  $('pw').placeholder = ''
  $('msg').textContent = r.ok ? 'Saved password forgotten.' : 'Could not forget the password.'
})
$('close').addEventListener('click', () => window.close())
