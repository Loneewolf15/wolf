<?php require APPROOT . '/views/inc/header.php'; ?>
<style>
    .row{
        background-color: black;
    }
    .card{
        background-color:goldenrod;
    }
    #row{
        background-color: goldenrod;
    }
    input{
        background: transparent;
    }
</style>
<div class="row mt-0">
    <div class="col-md-10 mx-auto">
        <div class="card card-body mt-5 mb-5">
            <h2>Speaking Request Form</h2>
            <p>Kindly fill this form to request for Sola's Speaking service</p>
            <form action="<?php echo URLROOT;?>/bookme/speakingform" method="post" class='p-2'>
               <div class="row" id="row">
                    <div class="form-group col-6 p-2">
                        <label for="name">Full name: <sup style="color: red;">*</sup></label>
                        <input type="text" name="name" class="form-control form-control-lg <?php echo (!empty($data['name_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['name'];?>">
                        <span class="invalid-feedback"><?php echo $data['name_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="email">Email: <sup style="color: red;">*</sup></label>
                        <input type="text" name="email" class="form-control form-control-lg <?php echo (!empty($data['email_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['email'];?>">
                        <span class="invalid-feedback"><?php echo $data['email_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="password">Password: <sup style="color: red;">*</sup></label>
                        <input type="password" name="password" class="form-control form-control-lg <?php echo (!empty($data['password_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['password'];?>">
                        <span class="invalid-feedback"><?php echo $data['password_err'];?></span>
                    </div> 
                    <div class="form-group col-6 p-2">
                        <label for="confirm_password">Confirm Password: <sup style="color: red;">*</sup></label>
                        <input type="password" name="confirm_password" class="form-control form-control-lg <?php echo (!empty($data['confirm_password_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['confirm_password'];?>">
                        <span class="invalid-feedback"><?php echo $data['confirm_password_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="date">Date of birth: <sup style="color: red;">*</sup></label>
                        <input type="date" name="date" class="form-control form-control-lg date <?php echo (!empty($data['date_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['date'];?>">
                        <span class="invalid-feedback"><?php echo $data['date_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="email">Phone number: <sup style="color: red;">*</sup></label>
                        <input type="number" name="number" class="form-control form-control-lg <?php echo (!empty($data['number_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['number'];?>">
                        <span class="invalid-feedback"><?php echo $data['number_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="status">Am interested In: <sup style="color: red;">*</sup></label>
                        <div class="">
                            <input type="radio" name="employed" value="Development for my people" class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Development for my people
                        </div>
                        <div class="">
                            <input type="radio" name="employed" value="Development for me"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Development for me
                        </div>
                        <div class="">
                            <input type="radio" name="employed" value="Booking Sola for my next event"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Booking Sola for my next event
                        </div>
                        <div class="">
                            <input type="radio" onclick="addValue()" name="other" value="Other" class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>">Other
                        </div>
                        <div class="valuee row bg-transparent p-2">
                            <input type="text" id="value" class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"   style="border-top: none; border-bottom:1px solid black; border-right:none;border-left:none;border-radius:3px; display: none;" placeholder="place interest">
                       </div> 
                        <span class="text-danger"><?php echo $data['employed_err'];?></span>
                    </div>                 
                    <div class="form-group col-6 p-2">
                        <label for="">Time duration<sup style="color: red;">*</sup></label>
                        <input type="text" name="status_duration" class="form-control form-control-lg <?php echo (!empty($data['status_duration_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['status_duration'];?>" >
                        <span class="invalid-feedback"><?php echo $data['status_duration_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="">State the theme/title of the event?: <sup style="color: red;">*</sup></label>
                        <input type="text" name="event_title" class="form-control form-control-lg <?php echo (!empty($data['event_title_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['event_title'];?>">
                        <span class="invalid-feedback"><?php echo $data['event_title_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="organizers">Who are the organizers of this event?: <sup style="color: red;">*</sup></label>
                        <input type="text" name="organizers" class="form-control form-control-lg <?php echo (!empty($data['organizers_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['organizers'];?>">
                        <span class="invalid-feedback"><?php echo $data['organizers_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                            <input type="submit" value="Submit" class="btn btn-success btn-block"  style="background-color:black">
                    </div>
                    <div class=" form-group col-6 p-2">
                            <a href="<?php echo URLROOT;?>" style="background-color:gold" class="btn btn-light btn-block">Cancel</a>
                    </div> 
               </div>
            </form>
            <p>
                Your answers will be reviewed and will be reached out to you shortly to discuss further and schedule a date.
                <h6>NOTE:</h6> that Speaking session comes at a standard fee.
            </p>
        </div>
    </div>
</div>

<?php require APPROOT . '/views/inc/footer.php'; ?>