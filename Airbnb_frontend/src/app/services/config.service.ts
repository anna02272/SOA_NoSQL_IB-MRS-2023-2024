import {Injectable} from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class ConfigService {

  private _auth_api_url = 'https://localhost:8080/api';
  private _acc_api_url = 'https://localhost:8083/api';
  private _profile_api_url = 'https://localhost:8084/api';
  private _rec_api_url = 'https://localhost:8085/api';
  private _notif_api_url = 'https://localhost:8086/api';
  private _resv_api_url = 'https://localhost:8082/api';
  private _rating_api_url = 'https://localhost:8087/api';


  private _auth_url = this._auth_api_url + '/auth';
  private _login_url = this._auth_url + '/login';
  private _register_url = this._auth_url + '/register';
  private _verifyEmail_url = this._auth_url + '/verifyEmail';
  private _resendVerification_url = this._auth_url + '/resendVerification';
  private _forgotPassword_url = this._auth_url + '/forgotPassword';
  private _resetPassword_url = this._auth_url + '/resetPassword';

  private _user_url = this._auth_api_url + '/users';
  private _current_user_url = this._user_url + '/currentUser';
  private _changePassword_url = this._user_url + '/changePassword';
  private _deleteProfile_url = this._user_url + '/delete';

  private _acc_url = this._acc_api_url + '/accommodations';

  private _resv_url = this._resv_api_url + '/reservations';
  private _create_resv_url = this._resv_api_url + '/reservations/create';

  private _rating_url = this._rating_api_url + '/rating';

  get login_url(): string {
    return this._login_url;
  }
  get register_url(): string {
    return this._register_url;
  }
  get verifyEmail_url(): string {
    return this._verifyEmail_url;
  }
  get resendVerification_url(): string {
    return this._resendVerification_url;
  }
  get forgotPassword_url(): string {
    return this._forgotPassword_url;
  }
  get resetPassword_url(): string {
    return this._resetPassword_url;
  }
  get currentUser_url(): string {
    return this._current_user_url;
  }

  get createReservation_url(): string {
    return this._create_resv_url;
  }
  
  get changePassword_url(): string {
    return this._changePassword_url;
  }
  get acc_url(): string {
    return this._acc_url;
  }
  get resv_url(): string {
    return this._resv_url;
  }
  get deleteProfile_url(): string {
    return this._deleteProfile_url;
  }
  get rating_url(): string {
    return this._rating_url;
  }
  }


