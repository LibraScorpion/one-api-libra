import React, { useCallback } from 'react';
import { GoogleOAuthProvider, GoogleLogin, useGoogleLogin } from '@react-oauth/google';
import { Button, Box, Alert, CircularProgress } from '@mui/material';
import GoogleIcon from '@mui/icons-material/Google';
import { useNavigate } from 'react-router-dom';
import { useDispatch } from 'react-redux';
import { loginSuccess } from '../store/authSlice';
import { api } from '../services/api';

// 从环境变量获取 Google Client ID
const GOOGLE_CLIENT_ID = process.env.REACT_APP_GOOGLE_CLIENT_ID || '';

interface GoogleAuthButtonProps {
  mode?: 'signin' | 'signup';
  onSuccess?: (user: any) => void;
  onError?: (error: any) => void;
}

// Google登录按钮组件
export const GoogleAuthButton: React.FC<GoogleAuthButtonProps> = ({
  mode = 'signin',
  onSuccess,
  onError,
}) => {
  const navigate = useNavigate();
  const dispatch = useDispatch();
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 处理Google登录成功
  const handleGoogleSuccess = useCallback(async (credentialResponse: any) => {
    setLoading(true);
    setError(null);

    try {
      // 发送Google token到后端验证
      const response = await api.post('/api/auth/google/token', {
        token: credentialResponse.credential,
      });

      if (response.data.success) {
        const { access_token, refresh_token, user } = response.data;

        // 保存tokens到localStorage
        localStorage.setItem('access_token', access_token);
        localStorage.setItem('refresh_token', refresh_token);

        // 更新Redux store
        dispatch(loginSuccess({
          user,
          accessToken: access_token,
          refreshToken: refresh_token,
        }));

        // 调用成功回调
        if (onSuccess) {
          onSuccess(user);
        }

        // 跳转到仪表盘
        navigate('/dashboard');
      }
    } catch (err: any) {
      const errorMsg = err.response?.data?.message || '登录失败，请重试';
      setError(errorMsg);

      if (onError) {
        onError(err);
      }
    } finally {
      setLoading(false);
    }
  }, [dispatch, navigate, onSuccess, onError]);

  // 处理Google登录错误
  const handleGoogleError = useCallback(() => {
    setError('Google登录失败，请检查网络连接');
    if (onError) {
      onError(new Error('Google login failed'));
    }
  }, [onError]);

  return (
    <Box sx={{ width: '100%' }}>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
        <Box sx={{ position: 'relative' }}>
          {loading && (
            <Box
              sx={{
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                backgroundColor: 'rgba(255, 255, 255, 0.8)',
                zIndex: 1,
              }}
            >
              <CircularProgress size={24} />
            </Box>
          )}

          <GoogleLogin
            onSuccess={handleGoogleSuccess}
            onError={handleGoogleError}
            text={mode === 'signin' ? 'signin_with' : 'signup_with'}
            theme="outline"
            size="large"
            width="100%"
            locale="zh-CN"
          />
        </Box>
      </GoogleOAuthProvider>
    </Box>
  );
};

// 自定义Google登录按钮（使用Material-UI样式）
export const CustomGoogleButton: React.FC<GoogleAuthButtonProps> = ({
  mode = 'signin',
  onSuccess,
  onError,
}) => {
  const navigate = useNavigate();
  const dispatch = useDispatch();
  const [loading, setLoading] = React.useState(false);

  // 使用Google OAuth Hook
  const googleLogin = useGoogleLogin({
    onSuccess: async (tokenResponse) => {
      setLoading(true);

      try {
        // 获取用户信息
        const userInfoResponse = await fetch(
          'https://www.googleapis.com/oauth2/v2/userinfo',
          {
            headers: {
              Authorization: `Bearer ${tokenResponse.access_token}`,
            },
          }
        );

        const userInfo = await userInfoResponse.json();

        // 发送到后端
        const response = await api.post('/api/auth/google/callback', {
          access_token: tokenResponse.access_token,
          user_info: userInfo,
        });

        if (response.data.success) {
          const { access_token, refresh_token, user } = response.data;

          // 保存tokens
          localStorage.setItem('access_token', access_token);
          localStorage.setItem('refresh_token', refresh_token);

          // 更新Redux
          dispatch(loginSuccess({
            user,
            accessToken: access_token,
            refreshToken: refresh_token,
          }));

          if (onSuccess) {
            onSuccess(user);
          }

          navigate('/dashboard');
        }
      } catch (error) {
        console.error('Login error:', error);
        if (onError) {
          onError(error);
        }
      } finally {
        setLoading(false);
      }
    },
    onError: () => {
      if (onError) {
        onError(new Error('Google login failed'));
      }
    },
  });

  return (
    <Button
      fullWidth
      variant="outlined"
      size="large"
      startIcon={<GoogleIcon />}
      onClick={() => googleLogin()}
      disabled={loading}
      sx={{
        borderColor: '#dadce0',
        color: '#3c4043',
        textTransform: 'none',
        fontSize: '14px',
        fontWeight: 500,
        py: 1.5,
        '&:hover': {
          backgroundColor: '#f8f9fa',
          borderColor: '#dadce0',
        },
      }}
    >
      {loading ? (
        <CircularProgress size={20} />
      ) : (
        mode === 'signin' ? '使用 Google 登录' : '使用 Google 注册'
      )}
    </Button>
  );
};

// Google账号关联组件（用于已登录用户）
export const GoogleAccountLink: React.FC = () => {
  const [linked, setLinked] = React.useState(false);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 关联Google账号
  const linkGoogleAccount = useGoogleLogin({
    onSuccess: async (tokenResponse) => {
      setLoading(true);
      setError(null);

      try {
        const response = await api.post('/api/auth/google/link', {
          access_token: tokenResponse.access_token,
        });

        if (response.data.success) {
          setLinked(true);
        }
      } catch (err: any) {
        setError(err.response?.data?.message || '关联失败');
      } finally {
        setLoading(false);
      }
    },
  });

  // 取消关联
  const unlinkGoogleAccount = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await api.delete('/api/auth/google/link');

      if (response.data.success) {
        setLinked(false);
      }
    } catch (err: any) {
      setError(err.response?.data?.message || '取消关联失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {linked ? (
        <Box>
          <Alert severity="success" sx={{ mb: 2 }}>
            已关联 Google 账号
          </Alert>
          <Button
            variant="outlined"
            color="error"
            onClick={unlinkGoogleAccount}
            disabled={loading}
            startIcon={loading ? <CircularProgress size={20} /> : <GoogleIcon />}
          >
            取消关联 Google 账号
          </Button>
        </Box>
      ) : (
        <Button
          variant="outlined"
          onClick={() => linkGoogleAccount()}
          disabled={loading}
          startIcon={loading ? <CircularProgress size={20} /> : <GoogleIcon />}
        >
          关联 Google 账号
        </Button>
      )}
    </Box>
  );
};

// Google OAuth配置Provider
export const GoogleAuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  if (!GOOGLE_CLIENT_ID) {
    console.error('Google Client ID is not configured');
    return <>{children}</>;
  }

  return (
    <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
      {children}
    </GoogleOAuthProvider>
  );
};

export default GoogleAuthButton;