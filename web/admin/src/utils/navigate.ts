// navigate.ts — 提供在 React 组件外部使用 router navigate 的能力
// 用于 axios 拦截器等非组件上下文中导航

let _navigate: ((path: string) => void) | null = null;

export function setNavigate(navigateFn: (path: string) => void) {
  _navigate = navigateFn;
}

export function navigateTo(path: string) {
  if (_navigate) {
    _navigate(path);
  } else {
    window.location.href = path;
  }
}
