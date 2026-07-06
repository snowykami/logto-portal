import {
  Activity,
  Bell,
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  ExternalLink,
  FolderOpen,
  GitBranch,
  HelpCircle,
  Home,
  KeyRound,
  LogOut,
  Menu,
  Moon,
  Package,
  RefreshCw,
  Save,
  Shield,
  Sun,
  UserRound,
  UsersRound,
  Workflow,
  X,
} from 'lucide-react';
import { clsx } from 'clsx';
import { FormEvent, ReactNode, useEffect, useMemo, useState } from 'react';
import { api, ApiError } from './lib/api';
import type { Announcement, AppCatalogItem, PortalData, User } from './types';
import { Badge } from './components/ui/badge';
import { Button } from './components/ui/button';
import { Card } from './components/ui/card';
import { useConfirm } from './components/ui/confirm-dialog';
import { Input } from './components/ui/input';

type Page = 'dashboard' | 'profile' | 'security' | 'applications' | 'organizations' | 'notifications' | 'help';
type Theme = 'light' | 'dark' | 'system';
type AccountCenterLinks = {
  profileUrl: string;
  securityUrl: string;
  emailUrl: string;
  phoneUrl: string;
  usernameUrl: string;
  passwordUrl: string;
  passkeyAddUrl: string;
  passkeyManageUrl: string;
  authenticatorAppUrl: string;
  backupCodesUrl: string;
};

type SupportInfo = {
  email: string;
  accountCenter: AccountCenterLinks;
};

const defaultAccountCenterLinks: AccountCenterLinks = {
  profileUrl: 'https://auth.liteyuki.org/account/profile',
  securityUrl: 'https://auth.liteyuki.org/account/security',
  emailUrl: 'https://auth.liteyuki.org/account/email',
  phoneUrl: 'https://auth.liteyuki.org/account/phone',
  usernameUrl: 'https://auth.liteyuki.org/account/username',
  passwordUrl: 'https://auth.liteyuki.org/account/password',
  passkeyAddUrl: 'https://auth.liteyuki.org/account/passkey/add',
  passkeyManageUrl: 'https://auth.liteyuki.org/account/passkey/manage',
  authenticatorAppUrl: 'https://auth.liteyuki.org/account/authenticator-app',
  backupCodesUrl: 'https://auth.liteyuki.org/account/backup-codes/manage',
};

const pages: Array<{ id: Page; label: string; href: string; icon: ReactNode }> = [
  { id: 'dashboard', label: '首页', href: '/', icon: <Home size={18} /> },
  { id: 'profile', label: '个人资料', href: '/profile', icon: <UserRound size={18} /> },
  { id: 'security', label: '安全中心', href: '/security', icon: <Shield size={18} /> },
  { id: 'applications', label: '应用入口', href: '/applications', icon: <Workflow size={18} /> },
  { id: 'organizations', label: '组织权限', href: '/organizations', icon: <UsersRound size={18} /> },
  { id: 'notifications', label: '公告通知', href: '/notifications', icon: <Bell size={18} /> },
  { id: 'help', label: '帮助支持', href: '/help', icon: <HelpCircle size={18} /> },
];

const iconMap: Record<string, ReactNode> = {
  activity: <Activity size={20} />,
  'folder-open': <FolderOpen size={20} />,
  'git-branch': <GitBranch size={20} />,
  package: <Package size={20} />,
  workflow: <Workflow size={20} />,
  'messages-square': <UsersRound size={20} />,
};

export function App() {
  const [page, setPage] = useState<Page>(pageFromPath(location.pathname));
  const [data, setData] = useState<PortalData | null>(null);
  const [supportInfo, setSupportInfo] = useState<SupportInfo>({
    email: 'contact@liteyuki.org',
    accountCenter: defaultAccountCenterLinks,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [theme, setTheme] = useState<Theme>(() => (localStorage.getItem('theme') as Theme | null) ?? 'system');

  useEffect(() => applyTheme(theme), [theme]);

  useEffect(() => {
    const onPopState = () => setPage(pageFromPath(location.pathname));
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, []);

  useEffect(() => {
    void loadData();
    void api<SupportInfo>('/api/support-info').then(setSupportInfo).catch(() => undefined);
  }, []);

  async function loadData() {
    setLoading(true);
    setError(null);
    try {
      const [me, apps, announcements] = await Promise.all([
        api<{ user: User }>('/api/me'),
        api<{ applications: AppCatalogItem[] }>('/api/app-catalog'),
        api<{ announcements: Announcement[] }>('/api/announcements'),
      ]);
      setData({
        user: me.user,
        applications: apps.applications,
        announcements: announcements.announcements,
      });
    } catch (reason) {
      if (!(reason instanceof ApiError && reason.status === 401)) {
        setError('暂时无法加载账号门户，请稍后重试。');
      }
    } finally {
      setLoading(false);
    }
  }

  function navigate(nextPage: Page) {
    const target = pages.find((item) => item.id === nextPage);
    if (!target) {
      return;
    }
    history.pushState(null, '', target.href);
    setPage(nextPage);
    setMenuOpen(false);
  }

  function selectTheme(nextTheme: Theme) {
    localStorage.setItem('theme', nextTheme);
    setTheme(nextTheme);
  }

  if (loading) {
    return <ShellPlaceholder />;
  }

  if (error || !data) {
    return (
      <FullPageState
        title="账号门户暂不可用"
        description={error ?? '请重新登录后再试。'}
        action={<Button variant="primary" onClick={() => window.location.assign('/auth/login')}>登录</Button>}
      />
    );
  }

  const currentPage = pages.find((item) => item.id === page) ?? pages[0];

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-30 border-b border-border bg-card/95 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <button className="flex min-w-0 items-center gap-3 text-left" onClick={() => navigate('dashboard')}>
            <div className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-primary text-primary-foreground">
              <KeyRound size={19} />
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold">Yuki ID Portal</div>
              <div className="truncate text-xs text-muted-foreground">{data.user.email || data.user.sub}</div>
            </div>
          </button>
          <div className="hidden items-center gap-2 md:flex">
            <ThemeSwitch theme={theme} onChange={selectTheme} />
            <Button variant="ghost" icon={<RefreshCw size={16} />} onClick={() => void loadData()}>刷新</Button>
            <LogoutButton global={false} />
          </div>
          <Button className="md:hidden" variant="ghost" icon={menuOpen ? <X size={20} /> : <Menu size={20} />} onClick={() => setMenuOpen((value) => !value)} aria-label="菜单" />
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl grid-cols-1 gap-0 md:grid-cols-[240px_1fr]">
        <aside className={clsx('border-b border-border bg-card md:block md:min-h-[calc(100vh-4rem)] md:border-b-0 md:border-r', menuOpen ? 'block' : 'hidden')}>
          <nav className="space-y-1 p-3">
            {pages.map((item) => (
              <button
                key={item.id}
                className={clsx(
                  'flex h-11 w-full items-center gap-3 rounded-md px-3 text-sm transition',
                  page === item.id ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
                onClick={() => navigate(item.id)}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            ))}
          </nav>
          <div className="border-t border-border p-3 md:hidden">
            <ThemeSwitch theme={theme} onChange={selectTheme} />
          </div>
        </aside>

        <main className="min-w-0 px-4 py-5 sm:px-6 lg:px-8">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h1 className="text-2xl font-semibold tracking-normal">{currentPage.label}</h1>
              <p className="mt-1 text-sm text-muted-foreground">{pageSubtitle(page)}</p>
            </div>
          </div>

          {page === 'dashboard' && <Dashboard data={data} navigate={navigate} />}
          {page === 'profile' && <Profile user={data.user} reload={loadData} />}
          {page === 'security' && <Security links={supportInfo.accountCenter} />}
          {page === 'applications' && <Applications apps={data.applications} />}
          {page === 'organizations' && <Organizations user={data.user} apps={data.applications} />}
          {page === 'notifications' && <Notifications announcements={data.announcements} />}
          {page === 'help' && <Help email={supportInfo.email} />}
        </main>
      </div>
    </div>
  );
}

function Dashboard({ data, navigate }: { data: PortalData; navigate: (page: Page) => void }) {
  const latest = data.announcements.slice(0, 3);
  const commonApps = data.applications.slice(0, 4);

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_340px]">
      <section className="space-y-4">
        <Card className="p-5">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <Avatar user={data.user} size="lg" />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="truncate text-xl font-semibold">{displayName(data.user)}</h2>
                {data.user.emailVerified && <Badge tone="ok">邮箱已验证</Badge>}
              </div>
              <p className="mt-1 truncate text-sm text-muted-foreground">{data.user.email || data.user.sub}</p>
            </div>
            <Button variant="primary" icon={<UserRound size={16} />} onClick={() => navigate('profile')}>资料</Button>
          </div>
        </Card>

        <div className="grid gap-4 sm:grid-cols-3">
          <Metric title="Roles" value={data.user.roles.length} />
          <Metric title="Organizations" value={data.user.organizations.length} />
          <Metric title="Org Roles" value={data.user.organizationRoles.length} />
        </div>

        <Card className="p-5">
          <SectionTitle title="常用应用" action={<Button variant="ghost" icon={<ChevronRight size={16} />} onClick={() => navigate('applications')}>全部</Button>} />
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            {commonApps.map((app) => <ApplicationTile key={app.id} app={app} />)}
          </div>
        </Card>
      </section>

      <section className="space-y-4">
        <Card className="p-5">
          <SectionTitle title="账号安全" />
          <div className="mt-4 space-y-3 text-sm">
            <StatusLine ok={data.user.emailVerified} label="邮箱验证状态" />
            <StatusLine ok={data.user.roles.length > 0} label="基础角色授权" />
            <StatusLine ok={data.user.sub.length > 0} label="Logto sub 标识" />
          </div>
        </Card>

        <Card className="p-5">
          <SectionTitle title="最新公告" action={<Button variant="ghost" icon={<ChevronRight size={16} />} onClick={() => navigate('notifications')}>全部</Button>} />
          <div className="mt-4 space-y-3">
            {latest.map((item) => <AnnouncementRow key={item.id} item={item} />)}
          </div>
        </Card>
      </section>
    </div>
  );
}

function Profile({ user, reload }: { user: User; reload: () => Promise<void> }) {
  const confirm = useConfirm();
  const [name, setName] = useState(user.name);
  const [username, setUsername] = useState(user.preferredUsername);
  const [picture, setPicture] = useState(user.picture);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const confirmed = await confirm({
      title: '确认修改资料',
      description: '昵称、用户名和头像会通过后端提交到 Logto Management API，并更新当前账号资料。',
      confirmText: '提交修改',
    });
    if (!confirmed) {
      return;
    }
    setSaving(true);
    setMessage(null);
    try {
      await api('/api/me/profile', {
        method: 'PATCH',
        body: JSON.stringify({
          name,
          preferredUsername: username,
          picture,
        }),
      });
      setMessage('资料已通过 Logto Management API 提交。');
      await reload();
    } catch {
      setMessage('资料更新失败，请检查 Logto Management API 权限配置。');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
      <Card className="p-5">
        <form className="space-y-4" onSubmit={(event) => void submit(event)}>
          <Field label="昵称">
            <Input value={name} onChange={(event) => setName(event.target.value)} />
          </Field>
          <Field label="用户名">
            <Input value={username} onChange={(event) => setUsername(event.target.value)} />
          </Field>
          <Field label="头像 URL">
            <Input value={picture} onChange={(event) => setPicture(event.target.value)} />
          </Field>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" variant="primary" icon={<Save size={16} />} disabled={saving}>{saving ? '保存中' : '保存'}</Button>
            {message && <span className="text-sm text-muted-foreground">{message}</span>}
          </div>
        </form>
      </Card>

      <Card className="p-5">
        <SectionTitle title="基础资料" />
        <div className="mt-4 space-y-3">
          <Info label="Sub" value={user.sub} />
          <Info label="邮箱" value={user.email || '未提供'} />
          <Info label="手机号" value="由 Logto Account API 管理" />
          <Info label="社交账号" value="由 Logto Account API 管理" />
        </div>
      </Card>
    </div>
  );
}

function Security({ links }: { links: AccountCenterLinks }) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="p-5">
        <SectionTitle title="安全操作" />
        <div className="mt-4 space-y-3">
          <ActionLine title="修改密码" href={links.passwordUrl} />
          <ActionLine title="安全中心" href={links.securityUrl} />
          <ActionLine title="Authenticator MFA" href={links.authenticatorAppUrl} />
          <ActionLine title="备份代码" href={links.backupCodesUrl} />
          <ActionLine title="Passkey 管理" href={links.passkeyManageUrl} />
        </div>
      </Card>
      <SessionsPanel />
    </div>
  );
}

function SessionsPanel() {
  const [payload, setPayload] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setError(null);
    try {
      setPayload(await api('/api/me/sessions'));
    } catch {
      setError('会话列表暂不可用，请检查 Account API 权限。');
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <Card className="p-5">
      <SectionTitle title="活跃会话" action={<Button variant="ghost" icon={<RefreshCw size={16} />} onClick={() => void load()}>刷新</Button>} />
      <div className="mt-4 rounded-md border border-border bg-muted p-3 text-xs text-muted-foreground">
        {error ? error : <pre className="max-h-80 overflow-auto whitespace-pre-wrap">{JSON.stringify(payload ?? {}, null, 2)}</pre>}
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <LogoutButton global={false} />
        <LogoutButton global />
      </div>
    </Card>
  );
}

function Applications({ apps }: { apps: AppCatalogItem[] }) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {apps.map((app) => <ApplicationTile key={app.id} app={app} />)}
    </div>
  );
}

function Organizations({ user, apps }: { user: User; apps: AppCatalogItem[] }) {
  const blockedApps = apps.filter((app) => !app.accessible);
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="p-5">
        <SectionTitle title="当前权限" />
        <TokenList title="Roles" values={user.roles} />
        <TokenList title="Organizations" values={user.organizations} />
        <TokenList title="Organization Roles" values={user.organizationRoles} />
      </Card>
      <Card className="p-5">
        <SectionTitle title="无法访问的应用" />
        <div className="mt-4 space-y-3">
          {blockedApps.length === 0 ? (
            <p className="text-sm text-muted-foreground">当前应用目录没有发现权限阻断。</p>
          ) : (
            blockedApps.map((app) => (
              <div key={app.id} className="rounded-md border border-border p-3">
                <div className="font-medium">{app.name}</div>
                <div className="mt-1 text-sm text-muted-foreground">{app.reasons.join(', ')}</div>
              </div>
            ))
          )}
        </div>
      </Card>
    </div>
  );
}

function Notifications({ announcements }: { announcements: Announcement[] }) {
  return (
    <div className="space-y-3">
      {announcements.map((item) => (
        <Card key={item.id} className="p-5">
          <AnnouncementRow item={item} />
        </Card>
      ))}
    </div>
  );
}

function Help({ email }: { email: string }) {
  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
      <Card className="p-5">
        <SectionTitle title="帮助主题" />
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          {['登录异常', '账号迁移', '权限申请', '安全设置'].map((item) => (
            <div key={item} className="rounded-md border border-border p-4">
              <div className="font-medium">{item}</div>
              <div className="mt-2 text-sm text-muted-foreground">联系支持时请附上账号 sub 与问题发生时间。</div>
            </div>
          ))}
        </div>
      </Card>
      <Card className="p-5">
        <SectionTitle title="联系支持" />
        <a className="mt-4 inline-flex items-center gap-2 text-primary" href={`mailto:${email}`}>
          {email}
          <ExternalLink size={15} />
        </a>
      </Card>
    </div>
  );
}

function ApplicationTile({ app }: { app: AppCatalogItem }) {
  return (
    <Card className="p-4">
      <div className="flex items-start gap-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md bg-muted text-primary">{iconMap[app.icon] ?? <Workflow size={20} />}</div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <h3 className="truncate font-semibold">{app.name}</h3>
            <Badge tone={app.accessible && app.url ? 'ok' : 'warn'}>{app.accessible && app.url ? '可访问' : '受限'}</Badge>
          </div>
          <p className="mt-2 min-h-10 text-sm text-muted-foreground">{app.description}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {(app.requiredRoles ?? []).slice(0, 2).map((role) => <Badge key={role}>{role}</Badge>)}
          </div>
          <a
            className={clsx('mt-4 inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium', app.accessible && app.url ? 'bg-primary text-primary-foreground' : 'pointer-events-none bg-muted text-muted-foreground')}
            href={app.url}
            target="_blank"
            rel="noreferrer"
          >
            {app.url ? '打开' : '未配置入口'}
            <ExternalLink size={15} />
          </a>
        </div>
      </div>
    </Card>
  );
}

function LogoutButton({ global }: { global: boolean }) {
  const confirm = useConfirm();

  async function run() {
    const confirmed = await confirm({
      title: global ? '退出 Yuki ID' : '退出当前门户',
      description: global
        ? '这会清除当前门户会话，并跳转到 Logto 退出 Yuki ID 中心登录态。'
        : '这只会清除当前门户的登录会话，不会退出 Yuki ID 中心。',
      confirmText: global ? '退出 Yuki ID' : '退出门户',
      variant: global ? 'danger' : 'default',
    });
    if (!confirmed) {
      return;
    }
    if (global) {
      const result = await api<{ redirectUrl: string }>('/api/me/logout-global', { method: 'POST' });
      window.location.assign(result.redirectUrl);
      return;
    }
    await api('/api/me/logout', { method: 'POST' });
    window.location.assign('/');
  }

  return (
    <Button variant={global ? 'danger' : 'secondary'} icon={<LogOut size={16} />} onClick={() => void run()}>
      {global ? '退出 Yuki ID' : '退出'}
    </Button>
  );
}

function ThemeSwitch({ theme, onChange }: { theme: Theme; onChange: (theme: Theme) => void }) {
  return (
    <div className="inline-flex rounded-md border border-border bg-card p-1">
      {(['light', 'dark', 'system'] as Theme[]).map((item) => (
        <button
          key={item}
          className={clsx('grid h-8 w-10 place-items-center rounded-md text-muted-foreground transition hover:bg-muted', theme === item && 'bg-primary text-primary-foreground')}
          onClick={() => onChange(item)}
          title={item}
        >
          {item === 'light' ? <Sun size={16} /> : item === 'dark' ? <Moon size={16} /> : <Activity size={16} />}
        </button>
      ))}
    </div>
  );
}

function SectionTitle({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <h2 className="text-base font-semibold">{title}</h2>
      {action}
    </div>
  );
}

function Avatar({ user, size = 'md' }: { user: User; size?: 'md' | 'lg' }) {
  const dimension = size === 'lg' ? 'h-16 w-16 text-xl' : 'h-10 w-10 text-sm';
  if (user.picture) {
    return <img className={clsx('rounded-md object-cover', dimension)} src={user.picture} alt="" />;
  }
  return <div className={clsx('grid shrink-0 place-items-center rounded-md bg-accent text-accent-foreground font-semibold', dimension)}>{displayName(user).slice(0, 1).toUpperCase()}</div>;
}

function Metric({ title, value }: { title: string; value: number }) {
  return (
    <Card className="p-4">
      <div className="text-sm text-muted-foreground">{title}</div>
      <div className="mt-2 text-3xl font-semibold">{value}</div>
    </Card>
  );
}

function StatusLine({ ok, label }: { ok: boolean; label: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      {ok ? <CheckCircle2 className="text-emerald-500" size={18} /> : <CircleAlert className="text-amber-500" size={18} />}
    </div>
  );
}

function AnnouncementRow({ item }: { item: Announcement }) {
  const tone = item.severity === 'warning' ? 'warn' : item.severity === 'critical' ? 'danger' : 'neutral';
  return (
    <div>
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="font-medium">{item.title}</h3>
        <Badge tone={tone}>{item.severity}</Badge>
        {item.pinned && <Badge tone="ok">置顶</Badge>}
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{item.content}</p>
    </div>
  );
}

function ActionLine({ title, href }: { title: string; href: string }) {
  return (
    <a className="flex items-center justify-between rounded-md border border-border p-3 hover:bg-muted" href={href} target="_blank" rel="noreferrer">
      <span className="font-medium">{title}</span>
      <ExternalLink size={16} />
    </a>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-medium">{label}</span>
      {children}
    </label>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md border border-border p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 break-words text-sm font-medium">{value}</div>
    </div>
  );
}

function TokenList({ title, values }: { title: string; values: string[] }) {
  return (
    <div className="mt-5">
      <div className="mb-2 text-sm font-medium">{title}</div>
      <div className="flex flex-wrap gap-2">
        {values.length > 0 ? values.map((value) => <Badge key={value}>{value}</Badge>) : <span className="text-sm text-muted-foreground">暂无</span>}
      </div>
    </div>
  );
}

function FullPageState({ title, description, action }: { title: string; description: string; action: ReactNode }) {
  return (
    <div className="grid min-h-screen place-items-center bg-background px-4">
      <Card className="w-full max-w-md p-6 text-center">
        <div className="mx-auto grid h-12 w-12 place-items-center rounded-md bg-muted text-primary">
          <KeyRound size={24} />
        </div>
        <h1 className="mt-4 text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        <div className="mt-5">{action}</div>
      </Card>
    </div>
  );
}

function ShellPlaceholder() {
  return (
    <div className="min-h-screen bg-background p-4 sm:p-6">
      <div className="mx-auto max-w-7xl space-y-4">
        <div className="h-16 animate-pulse rounded-lg bg-muted" />
        <div className="grid gap-4 md:grid-cols-[240px_1fr]">
          <div className="h-96 animate-pulse rounded-lg bg-muted" />
          <div className="h-96 animate-pulse rounded-lg bg-muted" />
        </div>
      </div>
    </div>
  );
}

function displayName(user: User) {
  return user.name || user.preferredUsername || user.email || user.sub;
}

function pageSubtitle(page: Page) {
  const copy: Record<Page, string> = {
    dashboard: '账号资料、安全状态、应用入口和公告汇总。',
    profile: '基础资料由后端通过 Logto Management API 更新。',
    security: '密码、MFA、社交账号和活跃会话。',
    applications: '按角色和组织展示轻雪应用访问状态。',
    organizations: '解释当前 roles、organizations 和 organization_roles。',
    notifications: '系统公告、迁移公告、维护公告和内测通知。',
    help: '登录异常、迁移说明、常见问题和支持入口。',
  };
  return copy[page];
}

function pageFromPath(pathname: string): Page {
  const match = pages.find((item) => item.href === pathname);
  return match?.id ?? 'dashboard';
}

function applyTheme(theme: Theme) {
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  document.documentElement.classList.toggle('dark', theme === 'dark' || (theme === 'system' && prefersDark));
}
