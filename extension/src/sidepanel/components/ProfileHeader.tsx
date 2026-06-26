import type { LocalProfile, SiteDetection } from "../../shared/types";

type Props = {
  profile: LocalProfile;
  site: SiteDetection | null;
  surveyId: string;
};

export function ProfileHeader({ profile, site, surveyId }: Props) {
  return (
    <header className="workspace-header">
      <div>
        <h1>问卷 AI 工作台</h1>
        <p>Survey AI Workspace</p>
      </div>
      <dl>
        <div>
          <dt>当前 Profile</dt>
          <dd>{profile.profileName || profile.profileKey || "未绑定"}</dd>
        </div>
        <div>
          <dt>当前网站</dt>
          <dd>{site?.site_name || site?.site_key || "未识别"}</dd>
        </div>
        <div>
          <dt>当前问卷</dt>
          <dd>{surveyId ? `#${surveyId}` : "未识别"}</dd>
        </div>
      </dl>
    </header>
  );
}
