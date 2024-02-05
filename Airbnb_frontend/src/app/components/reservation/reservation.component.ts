import { Component, Input, OnInit } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { ReservationService } from 'src/app/services/reservation.service';
import { Reservation } from 'src/app/models/reservation';
import { DatePipe } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { switchMap } from 'rxjs/operators';

@Component({
  selector: 'app-reservation',
  templateUrl: './reservation.component.html',
  styleUrls: ['./reservation.component.css']
})
export class ReservationComponent implements OnInit {
  @Input() accommodationId!: string;
  hostId!: string;
  hostFeatured!: boolean;
  hostEmail!: string;
  hostIdP!: string;
  form!: FormGroup;
  showDiv: boolean = false;
  showDivSuccess: boolean = false;
  showDivSuccessAvailability: boolean = false;
  check_in_date?: string;
  check_out_date?: string;
  check_in_time?: number;
  number_of_guests?: number;
  errorCheck: boolean = false;
  errorCheckGuests: boolean = false;
  errorMessage?: "";
  successMessage?: "Reserved successfully!"
  errorMessage2?: "Please enter your check-in time!"

  constructor(private fb: FormBuilder, private reservationService: ReservationService, private datePipe: DatePipe, private httpClient: HttpClient) {
    this.showDivSuccess = false;
  }

  ngOnInit(): void {
    this.form = this.fb.group({
      check_in_time: [''],
      check_out_date: [''],
      check_in_date: [''],
      number_of_guests: ['']
    });
//     this.httpClient.get('https://localhost:8000/api/accommodations/get/hostid/' + this.accommodationId).pipe(
//   switchMap((response: any) => {
//     this.hostId = response.hostId;
//     alert("accid " + this.accommodationId + " hostid " + this.hostId);
//     return this.httpClient.get('https://localhost:8000/api/users/getById/' + this.hostId);
//   }),
//   switchMap((response: any) => {
//     this.hostEmail = response.email;
//     alert("hostEmail " + this.hostEmail);
//     return this.httpClient.get('https://localhost:8000/api/profile/getUser/' + this.hostEmail);
//   }),
//   switchMap((response: any) => {
//     this.hostIdP = response.id;
//     alert("hostIdP " + this.hostIdP);
//     return this.httpClient.get('https://localhost:8000/api/profile/isFeatured/' + this.hostIdP);
//   })
// ).subscribe(
//   (response: any) => {
//     this.hostFeatured = response;
//     alert("hostFeatured " + this.hostFeatured);
//   },
//   error => {
//     console.error('Error', error);
//   }
// );
  }

  convertToISOFormat(dateObject?: string, isCheckOut?: boolean): string {
    const isoFormat = this.datePipe.transform(dateObject, 'yyyy-MM-ddTHH:mm:ss') + 'Z';

    // If it's a check-out date and not provided by the user, assume 15:00:00Z
    if (isCheckOut && !dateObject) {
      return isoFormat.replace(/00:00:00/, '15:00:00');
    }

    return isoFormat;
  }

  createReservation(): void {
    if (this.check_in_time === undefined) {
      this.errorCheck = true;
      return;
    } else {
      if (this.check_in_time > 24 || this.check_in_time < 1) {
        this.errorCheck = true;
        return;
      }
    }

    if (this.number_of_guests === undefined) {
      this.errorCheckGuests = true;
      return;
    } else {
      if (this.number_of_guests < 1) {
        this.errorCheckGuests = true;
        return;
      }
    }

    this.errorCheckGuests = false;
    this.errorCheck = false;

    const reservationCreate: Reservation = {
      accommodation_id: this.accommodationId,
      check_in_date: this.convertToISOFormat(this.check_in_date),
      check_out_date: this.convertToISOFormat(this.check_out_date, true), // Specify it's a check-out date
      number_of_guests: this.number_of_guests
    };

    this.reservationService.createReservation(reservationCreate).subscribe(
      {
        next: (response) => {
          console.log('Reservation created successfully', response);
          this.showDivSuccess = true;
          //this.isHostFeatured();
          setTimeout(() => {
            this.showDivSuccess = false;
          }, 5000);
        },
        error: (error) => {
          console.log(reservationCreate)
          this.showDiv = true;
          //this.isHostFeatured();
          this.errorMessage = error.error.error;
          setTimeout(() => {
            this.showDiv = false;
          }, 5000);
        }
      }
    );
  }

  checkAvailability(): void {
    this.errorCheck = false;

    const checkAvailabilityData = {
      check_in_date: this.convertToISOFormat(this.check_in_date),
      check_out_date: this.convertToISOFormat(this.check_out_date, true), // Specify it's a check-out date
    };

    this.reservationService.checkAvailability(checkAvailabilityData, this.accommodationId).subscribe(
      {
        next: (response) => {
          console.log('Dates are available.', response);
          this.showDivSuccessAvailability = true;
          setTimeout(() => {
            this.showDivSuccessAvailability = false;
          }, 5000);
        },
        error: (error) => {
          this.showDiv = true;
          this.errorMessage = error.error.error;
          console.log(error);
          setTimeout(() => {
            this.showDiv = false;
          }, 5000);
        }
      }
    );
  }

  // isHostFeatured() {
  //   alert("isHostFeatured");
  //   var featured = false;
    
  //   // var averageRating = 0;
  //   // this.ratingService.getAll().subscribe(
  //   //   (response: any) => {
  //   //     averageRating = response.averageRating;
  //   //   },
  //   //   error => {
  //   //     console.error('Error fetching ratings', error);
  //   //   }
  //   // );
  //   // if (averageRating >= 4.7) {
  //   //   featured = true;
  //   // }

  //   var cancelRate = 0;
  //   this.httpClient.get('https://localhost:8000/api/reservations/cancelled/' + this.hostId).subscribe(
  //     (response: any) => {
  //       cancelRate = response;
  //       alert("cancel rate " + cancelRate);
  //     },
  //     error => {
  //       console.error('Error fetching cancel rate', error);
  //     }
  //   );
  //   if (cancelRate < 5.0) {
  //     featured = true;
  //   }

  //   var total = 0;
  //   this.httpClient.get('https://localhost:8000/api/reservations/total/' + this.hostId).subscribe(
  //     (response: any) => {
  //       total = response;
  //       alert("total " + total);
  //     },
  //     error => {
  //       console.error('Error fetching total', error);
  //     }
  //   );
  //   if (total >= 5) {
  //     featured = true;
  //   }

  //   var duration = 0;
  //   this.httpClient.get('https://localhost:8000/api/reservations/duration/' + this.hostId).subscribe(
  //     (response: any) => {
  //       duration = response;
  //       alert("duration " + duration);
  //     },
  //     error => {
  //       console.error('Error fetching duration', error);
  //     }
  //   );
  //   if (duration > 50) {
  //     featured = true;
  //   }

  //   var responseFeatured = false;
    
  //   if (this.hostFeatured) {
  //     if (!responseFeatured) {
  //       //post to https://localhost:8000/api/hosts/featured/{hostId}
  //       this.httpClient.post('https://localhost:8000/api/profile/setFeatured/' + this.hostId, null).subscribe(
  //         (response: any) => {
  //           console.log(response);
  //           alert("response set featured " + response);
  //         },
  //         error => {
  //           console.error('Error featuring host', error);
  //           alert("error set featured " + error);
  //         }
  //       );
  //     }
  //   } else{
  //     if (responseFeatured) {
  //       this.httpClient.post('https://localhost:8000/api/profile/setUnfeatured/' + this.hostId, null).subscribe(
  //         (response: any) => {
  //           console.log(response);
  //           alert("response set unfeatured " + response);
  //         },
  //         error => {
  //           console.error('Error removing feature from host', error);
  //           alert("error set unfeatured " + error);
  //         }
  //       );
  //     }
  //   }

  // }
  
}

