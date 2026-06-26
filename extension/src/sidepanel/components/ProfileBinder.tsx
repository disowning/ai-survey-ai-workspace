import { Link, Save } from "lucide-react";
import type { FormEvent } from "react";
import type { LocalProfile } from "../../shared/types";

type Props = {
  profile: LocalProfile;
  apiBaseUrl: string;
  busy: boolean;
  onApiBaseUrlChange: (value: string) => void;
  onProfileDraftChange: (profile: LocalProfile) => void;
  onBind: () => void;
};

export function ProfileBinder({
  profile,
  apiBaseUrl,
  busy,
  onApiBaseUrlChange,
  onProfileDraftChange,
  onBind
}: Props) {
  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    onBind();
  }

  return (
    <section className="panel-section">
      <div className="section-title">
        <Link size={18} aria-hidden="true" />
        <h2>绑定 Profile</h2>
      </div>
      <form className="stack" onSubmit={handleSubmit}>
        <label>
          <span>后端地址</span>
          <input value={apiBaseUrl} onChange={(event) => onApiBaseUrlChange(event.target.value)} />
        </label>
        <label>
          <span>Profile Key</span>
          <input
            value={profile.profileKey}
            placeholder="profile-001"
            onChange={(event) => onProfileDraftChange({ ...profile, profileKey: event.target.value })}
          />
        </label>
        <label>
          <span>Profile 名称</span>
          <input
            value={profile.profileName}
            placeholder="Profile-001"
            onChange={(event) => onProfileDraftChange({ ...profile, profileName: event.target.value })}
          />
        </label>
        <button className="primary-button" type="submit" disabled={busy || !profile.profileKey.trim()}>
          <Save size={16} aria-hidden="true" />
          绑定 Profile
        </button>
      </form>
    </section>
  );
}
