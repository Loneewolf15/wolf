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
            <h2>Coaching Request Form</h2>
            <p>Kindly fill this form to request for Sola's Coaching service</p>
            <form action="<?php echo URLROOT;?>/bookme/coachingform" method="post" class='p-2'>
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
                        <label for="status">Employment status: <sup style="color: red;">*</sup></label>
                        <div class="">
                            <input type="radio" name="employed" value="Employed"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Employed
                        </div>
                        <div class="">
                            <input type="radio" name="employed" value="Unemployed"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Unemployed
                        </div>
                        <div class="">
                            <input type="radio" name="employed" value="Self-Employed"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Self-Employed
                        </div>
                        <div class="">
                            <input type="radio" onclick="addValue()" name="employed" value="Other"class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"> Other
                        </div>  
                        <div class="valuee row bg-transparent p-2">
                            <input type="text" id="value"  class="<?php echo (!empty($data['employed_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['employed'];?>"   style="border-top: none; border-bottom:1px solid black; border-right:none;border-left:none;border-radius:3px; display: none;" placeholder="add employment status">
                       </div>                
                        <span class="invalid-feedback"><?php echo $data['employed_err'];?></span>
                    </div> 
                    <div class="form-group col-6 p-2">
                        <label for=""> How long have you been on your Employent status?: <sup style="color: red;">*</sup></label>
                        <input type="text" name="status_duration" class="form-control form-control-lg <?php echo (!empty($data['status_duration_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['status_duration'];?>" >
                        <span class="invalid-feedback"><?php echo $data['status_duration_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for=""> Why do you want Sola to Coach you?: <sup style="color: red;">*</sup></label>
                        <input type="text" name="coaching_reason" class="form-control form-control-lg <?php echo (!empty($data['coaching_reason_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['coaching_reason'];?>">
                        <span class="invalid-feedback"><?php echo $data['coaching_reason_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                        <label for="expectation"> What are your expectations: <sup style="color: red;">*</sup></label>
                        <input type="text" name="expectation" class="form-control form-control-lg <?php echo (!empty($data['expectation_err'])) ? 'is-invalid' : '';?>" value="<?php echo $data['expectation'];?>">
                        <span class="invalid-feedback"><?php echo $data['expectation_err'];?></span>
                    </div>
                    <div class="form-group col-6 p-2">
                            <input type="submit" value="Submit" style="background-color:black" class="btn btn-success btn-block">
                    </div>
                    <div class=" form-group col-6 p-2">
                            <a href="<?php echo URLROOT;?>" style="background-color:gold" class="btn btn-light btn-block">Cancel</a>
                    </div> 
               </div>
            </form>
            <p>
                Your answers will be reviewed and will be reached out to you shortly to discuss further and schedule a date.
                <h6>NOTE:</h6> that Coaching session comes at a standard fee.
            </p>
        </div>
    </div>
</div>

<?php require APPROOT . '/views/inc/footer.php'; ?>